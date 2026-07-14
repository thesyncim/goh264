// SPDX-License-Identifier: LGPL-2.1-or-later

#include <errno.h>
#include <inttypes.h>
#include <pthread.h>
#include <stdarg.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

#include <libavcodec/avcodec.h>
#include <libavutil/cpu.h>
#include <libavutil/error.h>
#include <libavutil/log.h>

#if defined(__OPTIMIZE__)
#define GOH264_HELPER_OPTIMIZED "true"
#else
#define GOH264_HELPER_OPTIMIZED "false"
#endif

typedef struct Input {
    uint8_t *data;
    size_t size;
} Input;

typedef struct Worker {
    const Input *input;
    const AVCodec *codec;
    AVCodecContext *decoder;
    AVCodecParserContext *parser;
    AVPacket *packet;
    AVFrame *frame;
    int iterations;
    int64_t frames;
    int error;
    char error_text[AV_ERROR_MAX_STRING_SIZE];
} Worker;

typedef struct StartGate {
    pthread_mutex_t mutex;
    pthread_cond_t cond;
    int ready;
    bool start;
} StartGate;

typedef struct ThreadTask {
    Worker *worker;
    StartGate *gate;
} ThreadTask;

typedef struct Options {
    const char *input;
    int iterations;
    int repeats;
    int warmup;
    int workers;
    bool pure_c;
} Options;

static void usage(const char *program) {
    fprintf(stderr,
            "usage: %s --input FILE [--iters N] [--repeats N] [--warmup N] "
            "[--workers N] [--cpu-flags native|0]\n",
            program);
}

static int parse_positive(const char *name, const char *value, bool allow_zero) {
    char *end = NULL;
    errno = 0;
    long parsed = strtol(value, &end, 10);
    if (errno != 0 || end == value || *end != '\0' || parsed > INT32_MAX ||
        parsed < (allow_zero ? 0 : 1)) {
        fprintf(stderr, "%s: invalid value %s\n", name, value);
        return -1;
    }
    return (int)parsed;
}

static int parse_options(int argc, char **argv, Options *options) {
    *options = (Options){
        .iterations = 1,
        .repeats = 1,
        .warmup = 1,
        .workers = 1,
    };
    for (int i = 1; i < argc; i++) {
        if (strcmp(argv[i], "--input") == 0 && i + 1 < argc) {
            options->input = argv[++i];
        } else if (strcmp(argv[i], "--iters") == 0 && i + 1 < argc) {
            options->iterations = parse_positive("--iters", argv[++i], false);
        } else if (strcmp(argv[i], "--repeats") == 0 && i + 1 < argc) {
            options->repeats = parse_positive("--repeats", argv[++i], false);
        } else if (strcmp(argv[i], "--warmup") == 0 && i + 1 < argc) {
            options->warmup = parse_positive("--warmup", argv[++i], true);
        } else if (strcmp(argv[i], "--workers") == 0 && i + 1 < argc) {
            options->workers = parse_positive("--workers", argv[++i], false);
        } else if (strcmp(argv[i], "--cpu-flags") == 0 && i + 1 < argc) {
            const char *flags = argv[++i];
            if (strcmp(flags, "0") == 0) {
                options->pure_c = true;
            } else if (strcmp(flags, "native") != 0) {
                fprintf(stderr, "--cpu-flags: expected native or 0, got %s\n", flags);
                return -1;
            }
        } else {
            fprintf(stderr, "unknown or incomplete option: %s\n", argv[i]);
            return -1;
        }
    }
    if (options->input == NULL || options->iterations < 1 || options->repeats < 1 ||
        options->warmup < 0 || options->workers < 1) {
        return -1;
    }
    return 0;
}

static int load_input(const char *path, Input *input) {
    FILE *file = fopen(path, "rb");
    if (file == NULL) {
        fprintf(stderr, "%s: %s\n", path, strerror(errno));
        return -1;
    }
    if (fseek(file, 0, SEEK_END) != 0) {
        fprintf(stderr, "%s: seek: %s\n", path, strerror(errno));
        fclose(file);
        return -1;
    }
    long size = ftell(file);
    if (size < 0 || fseek(file, 0, SEEK_SET) != 0) {
        fprintf(stderr, "%s: size/rewind: %s\n", path, strerror(errno));
        fclose(file);
        return -1;
    }
    if ((uint64_t)size > SIZE_MAX - AV_INPUT_BUFFER_PADDING_SIZE) {
        fprintf(stderr, "%s: input is too large\n", path);
        fclose(file);
        return -1;
    }
    uint8_t *data = calloc((size_t)size + AV_INPUT_BUFFER_PADDING_SIZE, 1);
    if (data == NULL) {
        fprintf(stderr, "%s: allocate input: %s\n", path, strerror(errno));
        fclose(file);
        return -1;
    }
    if (size > 0 && fread(data, 1, (size_t)size, file) != (size_t)size) {
        fprintf(stderr, "%s: read: %s\n", path, ferror(file) ? strerror(errno) : "short read");
        free(data);
        fclose(file);
        return -1;
    }
    fclose(file);
    input->data = data;
    input->size = (size_t)size;
    return 0;
}

static void worker_error(Worker *worker, int error, const char *operation) {
    worker->error = error;
    char detail[AV_ERROR_MAX_STRING_SIZE];
    av_strerror(error, detail, sizeof(detail));
    snprintf(worker->error_text, sizeof(worker->error_text), "%s: %s", operation, detail);
}

static int worker_receive_frames(Worker *worker, int64_t *frames) {
    for (;;) {
        int ret = avcodec_receive_frame(worker->decoder, worker->frame);
        if (ret == AVERROR(EAGAIN) || ret == AVERROR_EOF) {
            return 0;
        }
        if (ret < 0) {
            worker_error(worker, ret, "avcodec_receive_frame");
            return ret;
        }
        (*frames)++;
        av_frame_unref(worker->frame);
    }
}

static int worker_send_packet(Worker *worker, AVPacket *packet, int64_t *frames) {
    int ret = avcodec_send_packet(worker->decoder, packet);
    if (ret == AVERROR(EAGAIN)) {
        ret = worker_receive_frames(worker, frames);
        if (ret < 0) {
            return ret;
        }
        ret = avcodec_send_packet(worker->decoder, packet);
    }
    if (ret < 0) {
        worker_error(worker, ret, "avcodec_send_packet");
        return ret;
    }
    return worker_receive_frames(worker, frames);
}

static int worker_parse(Worker *worker, const uint8_t *data, size_t size, int64_t *frames) {
    while (size > 0) {
        uint8_t *packet_data = NULL;
        int packet_size = 0;
        int chunk = size > INT32_MAX ? INT32_MAX : (int)size;
        int consumed = av_parser_parse2(worker->parser, worker->decoder,
                                        &packet_data, &packet_size,
                                        data, chunk,
                                        AV_NOPTS_VALUE, AV_NOPTS_VALUE, 0);
        if (consumed < 0) {
            worker_error(worker, consumed, "av_parser_parse2");
            return consumed;
        }
        if (consumed == 0 && packet_size == 0) {
            worker_error(worker, AVERROR_INVALIDDATA, "av_parser_parse2 made no progress");
            return AVERROR_INVALIDDATA;
        }
        data += consumed;
        size -= (size_t)consumed;
        if (packet_size == 0) {
            continue;
        }
        worker->packet->data = packet_data;
        worker->packet->size = packet_size;
        int ret = worker_send_packet(worker, worker->packet, frames);
        av_packet_unref(worker->packet);
        if (ret < 0) {
            return ret;
        }
    }
    return 0;
}

static int worker_flush_parser(Worker *worker, int64_t *frames) {
    for (;;) {
        uint8_t *packet_data = NULL;
        int packet_size = 0;
        int ret = av_parser_parse2(worker->parser, worker->decoder,
                                   &packet_data, &packet_size,
                                   NULL, 0,
                                   AV_NOPTS_VALUE, AV_NOPTS_VALUE, 0);
        if (ret < 0) {
            worker_error(worker, ret, "av_parser_parse2 flush");
            return ret;
        }
        if (packet_size == 0) {
            return 0;
        }
        worker->packet->data = packet_data;
        worker->packet->size = packet_size;
        ret = worker_send_packet(worker, worker->packet, frames);
        av_packet_unref(worker->packet);
        if (ret < 0) {
            return ret;
        }
    }
}

static int worker_reset(Worker *worker) {
    avcodec_flush_buffers(worker->decoder);
    av_parser_close(worker->parser);
    worker->parser = av_parser_init(AV_CODEC_ID_H264);
    if (worker->parser == NULL) {
        worker_error(worker, AVERROR(ENOMEM), "av_parser_init");
        return AVERROR(ENOMEM);
    }
    return 0;
}

static int worker_iteration(Worker *worker, int64_t *frames) {
    int ret = worker_parse(worker, worker->input->data, worker->input->size, frames);
    if (ret >= 0) {
        ret = worker_flush_parser(worker, frames);
    }
    if (ret >= 0) {
        ret = worker_send_packet(worker, NULL, frames);
    }
    if (ret >= 0) {
        ret = worker_reset(worker);
    }
    return ret;
}

static void worker_run(Worker *worker) {
    worker->frames = 0;
    worker->error = 0;
    worker->error_text[0] = '\0';
    for (int i = 0; i < worker->iterations; i++) {
        if (worker_iteration(worker, &worker->frames) < 0) {
            return;
        }
    }
}

static int worker_init(Worker *worker, const Input *input, const AVCodec *codec) {
    memset(worker, 0, sizeof(*worker));
    worker->input = input;
    worker->codec = codec;
    worker->decoder = avcodec_alloc_context3(codec);
    worker->parser = av_parser_init(AV_CODEC_ID_H264);
    worker->packet = av_packet_alloc();
    worker->frame = av_frame_alloc();
    if (worker->decoder == NULL || worker->parser == NULL ||
        worker->packet == NULL || worker->frame == NULL) {
        worker_error(worker, AVERROR(ENOMEM), "allocate decoder state");
        return -1;
    }
    worker->decoder->thread_count = 1;
    worker->decoder->thread_type = 0;
    int ret = avcodec_open2(worker->decoder, codec, NULL);
    if (ret < 0) {
        worker_error(worker, ret, "avcodec_open2");
        return -1;
    }
    return 0;
}

static void worker_close(Worker *worker) {
    if (worker->parser != NULL) {
        av_parser_close(worker->parser);
    }
    avcodec_free_context(&worker->decoder);
    av_packet_free(&worker->packet);
    av_frame_free(&worker->frame);
}

static void *thread_main(void *opaque) {
    ThreadTask *task = opaque;
    pthread_mutex_lock(&task->gate->mutex);
    task->gate->ready++;
    pthread_cond_broadcast(&task->gate->cond);
    while (!task->gate->start) {
        pthread_cond_wait(&task->gate->cond, &task->gate->mutex);
    }
    pthread_mutex_unlock(&task->gate->mutex);
    worker_run(task->worker);
    return NULL;
}

static double monotonic_seconds(void) {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return (double)ts.tv_sec + (double)ts.tv_nsec / 1000000000.0;
}

static int run_sample(Worker *workers, int worker_count, int iterations,
                      double *elapsed_ms, int64_t *total_frames) {
    pthread_t *threads = calloc((size_t)worker_count, sizeof(*threads));
    ThreadTask *tasks = calloc((size_t)worker_count, sizeof(*tasks));
    StartGate gate = {
        .mutex = PTHREAD_MUTEX_INITIALIZER,
        .cond = PTHREAD_COND_INITIALIZER,
    };
    if (threads == NULL || tasks == NULL) {
        fprintf(stderr, "allocate thread state: %s\n", strerror(errno));
        free(threads);
        free(tasks);
        return -1;
    }

    int created = 0;
    for (int i = 0; i < worker_count; i++) {
        workers[i].iterations = iterations;
        tasks[i] = (ThreadTask){.worker = &workers[i], .gate = &gate};
        int ret = pthread_create(&threads[i], NULL, thread_main, &tasks[i]);
        if (ret != 0) {
            fprintf(stderr, "pthread_create: %s\n", strerror(ret));
            pthread_mutex_lock(&gate.mutex);
            gate.start = true;
            pthread_cond_broadcast(&gate.cond);
            pthread_mutex_unlock(&gate.mutex);
            break;
        }
        created++;
    }
    if (created != worker_count) {
        for (int i = 0; i < created; i++) {
            pthread_join(threads[i], NULL);
        }
        pthread_cond_destroy(&gate.cond);
        pthread_mutex_destroy(&gate.mutex);
        free(threads);
        free(tasks);
        return -1;
    }

    pthread_mutex_lock(&gate.mutex);
    while (gate.ready != worker_count) {
        pthread_cond_wait(&gate.cond, &gate.mutex);
    }
    double start = monotonic_seconds();
    gate.start = true;
    pthread_cond_broadcast(&gate.cond);
    pthread_mutex_unlock(&gate.mutex);

    int status = 0;
    *total_frames = 0;
    for (int i = 0; i < worker_count; i++) {
        pthread_join(threads[i], NULL);
        *total_frames += workers[i].frames;
        if (workers[i].error < 0) {
            fprintf(stderr, "worker %d: %s\n", i, workers[i].error_text);
            status = -1;
        }
    }
    double end = monotonic_seconds();
    *elapsed_ms = (end - start) * 1000.0;

    pthread_cond_destroy(&gate.cond);
    pthread_mutex_destroy(&gate.mutex);
    free(threads);
    free(tasks);
    return status;
}

int main(int argc, char **argv) {
    Options options;
    if (parse_options(argc, argv, &options) < 0) {
        usage(argv[0]);
        return 2;
    }
    av_log_set_level(AV_LOG_ERROR);
    if (options.pure_c) {
        av_force_cpu_flags(0);
    }

    Input input = {0};
    if (load_input(options.input, &input) < 0) {
        return 1;
    }
    const AVCodec *codec = avcodec_find_decoder(AV_CODEC_ID_H264);
    if (codec == NULL) {
        fprintf(stderr, "H.264 decoder is unavailable\n");
        free(input.data);
        return 1;
    }
    Worker *workers = calloc((size_t)options.workers, sizeof(*workers));
    if (workers == NULL) {
        fprintf(stderr, "allocate workers: %s\n", strerror(errno));
        free(input.data);
        return 1;
    }
    int initialized = 0;
    for (; initialized < options.workers; initialized++) {
        if (worker_init(&workers[initialized], &input, codec) < 0) {
            fprintf(stderr, "worker %d: %s\n", initialized, workers[initialized].error_text);
            break;
        }
    }
    if (initialized != options.workers) {
        for (int i = 0; i <= initialized && i < options.workers; i++) {
            worker_close(&workers[i]);
        }
        free(workers);
        free(input.data);
        return 1;
    }

    if (options.warmup > 0) {
        double ignored_ms;
        int64_t ignored_frames;
        if (run_sample(workers, options.workers, options.warmup,
                       &ignored_ms, &ignored_frames) < 0) {
            for (int i = 0; i < options.workers; i++) {
                worker_close(&workers[i]);
            }
            free(workers);
            free(input.data);
            return 1;
        }
    }

    printf("{\"version\":2,\"backend\":\"libavcodec\",\"codec\":\"%s\","
           "\"libavcodec_version\":%u,\"cpu_flags\":\"%s\","
           "\"cpu_flags_mask\":%d,\"compiler\":\"%s\",\"optimized\":" GOH264_HELPER_OPTIMIZED ","
           "\"workers\":%d,\"decoder_threads_per_worker\":1,"
           "\"iterations_per_worker\":%d,\"repeats\":%d,\"samples\":[",
           codec->name, avcodec_version(), options.pure_c ? "0" : "native",
           av_get_cpu_flags(), __VERSION__, options.workers,
           options.iterations, options.repeats);
    int status = 0;
    for (int repeat = 0; repeat < options.repeats; repeat++) {
        double elapsed_ms;
        int64_t total_frames;
        if (run_sample(workers, options.workers, options.iterations,
                       &elapsed_ms, &total_frames) < 0) {
            status = 1;
            break;
        }
        printf("%s{\"elapsed_ms\":%.6f,\"total_frames\":%" PRId64 "}",
               repeat == 0 ? "" : ",", elapsed_ms, total_frames);
    }
    printf("]}\n");

    for (int i = 0; i < options.workers; i++) {
        worker_close(&workers[i]);
    }
    free(workers);
    free(input.data);
    return status;
}
