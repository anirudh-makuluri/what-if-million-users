package com.example.jobqueue.service;

import java.time.Instant;
import java.util.List;
import java.util.Map;

import com.example.jobqueue.model.JobStatus;
import com.example.jobqueue.persistence.JobEntity;

import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.core.script.DefaultRedisScript;
import org.springframework.stereotype.Component;

@Component
public class RedisQueueStore {

    private static final String READY_KEY = "job-queue:ready";
    private static final String PROCESSING_KEY = "job-queue:processing";
    private static final String STATE_PREFIX = "job-queue:state:";

    private static final DefaultRedisScript<Long> CLAIM_SCRIPT = new DefaultRedisScript<>(
            """
            local ready = KEYS[1]
            local processing = KEYS[2]
            local stateKey = KEYS[3]
            local jobId = ARGV[1]
            local now = tonumber(ARGV[2])
            local leaseUntil = tonumber(ARGV[3])

            local score = redis.call('ZSCORE', ready, jobId)
            if not score then
              return 0
            end

            if tonumber(score) > now then
              return 0
            end

            if redis.call('ZREM', ready, jobId) == 0 then
              return 0
            end

            redis.call('ZADD', processing, leaseUntil, jobId)
            redis.call('HSET', stateKey,
              'status', 'PROCESSING',
              'lockExpiresAt', leaseUntil,
              'updatedAt', now
            )
            return 1
            """,
            Long.class
    );

    private final StringRedisTemplate redisTemplate;

    public RedisQueueStore(StringRedisTemplate redisTemplate) {
        this.redisTemplate = redisTemplate;
    }

    public void enqueue(JobEntity job) {
        redisTemplate.executePipelined((org.springframework.data.redis.core.RedisCallback<Object>) connection -> {
            byte[] readyKey = redisTemplate.getStringSerializer().serialize(READY_KEY);
            byte[] stateKey = redisTemplate.getStringSerializer().serialize(stateKey(job.getId()));
            byte[] jobId = redisTemplate.getStringSerializer().serialize(job.getId());

            connection.zAdd(readyKey, toMillis(job.getAvailableAt()), jobId);
            connection.hMSet(stateKey, serializeState(job));
            connection.zRem(redisTemplate.getStringSerializer().serialize(PROCESSING_KEY), jobId);
            return null;
        });
    }

    public boolean claim(String jobId, Instant now, Instant leaseUntil) {
        Long result = redisTemplate.execute(
                CLAIM_SCRIPT,
                List.of(READY_KEY, PROCESSING_KEY, stateKey(jobId)),
                jobId,
                String.valueOf(toMillis(now)),
                String.valueOf(toMillis(leaseUntil))
        );
        return result != null && result == 1L;
    }

    public void markCompleted(JobEntity job) {
        redisTemplate.executePipelined((org.springframework.data.redis.core.RedisCallback<Object>) connection -> {
            byte[] jobId = redisTemplate.getStringSerializer().serialize(job.getId());
            connection.zRem(redisTemplate.getStringSerializer().serialize(READY_KEY), jobId);
            connection.zRem(redisTemplate.getStringSerializer().serialize(PROCESSING_KEY), jobId);
            connection.hMSet(redisTemplate.getStringSerializer().serialize(stateKey(job.getId())), serializeState(job));
            return null;
        });
    }

    public void markFailed(JobEntity job) {
        redisTemplate.executePipelined((org.springframework.data.redis.core.RedisCallback<Object>) connection -> {
            byte[] jobId = redisTemplate.getStringSerializer().serialize(job.getId());
            connection.zRem(redisTemplate.getStringSerializer().serialize(PROCESSING_KEY), jobId);
            connection.zRem(redisTemplate.getStringSerializer().serialize(READY_KEY), jobId);
            if (job.getStatus() == JobStatus.QUEUED) {
                connection.zAdd(
                        redisTemplate.getStringSerializer().serialize(READY_KEY),
                        toMillis(job.getAvailableAt()),
                        jobId
                );
            }
            connection.hMSet(redisTemplate.getStringSerializer().serialize(stateKey(job.getId())), serializeState(job));
            return null;
        });
    }

    public void restoreQueuedState(String jobId, Instant availableAt) {
        redisTemplate.opsForZSet().add(READY_KEY, jobId, toMillis(availableAt));
        redisTemplate.opsForZSet().remove(PROCESSING_KEY, jobId);
    }

    public void cacheState(JobEntity job) {
        redisTemplate.opsForHash().putAll(stateKey(job.getId()), toStateMap(job));
    }

    private Map<byte[], byte[]> serializeState(JobEntity job) {
        return toStateMap(job).entrySet().stream().collect(
                java.util.stream.Collectors.toMap(
                        entry -> redisTemplate.getStringSerializer().serialize(entry.getKey()),
                        entry -> redisTemplate.getStringSerializer().serialize(entry.getValue())
                )
        );
    }

    private Map<String, String> toStateMap(JobEntity job) {
        return Map.ofEntries(
                Map.entry("status", job.getStatus().name()),
                Map.entry("retryCount", String.valueOf(job.getRetryCount())),
                Map.entry("maxRetries", String.valueOf(job.getMaxRetries())),
                Map.entry("availableAt", String.valueOf(toMillis(job.getAvailableAt()))),
                Map.entry("processorId", nullToEmpty(job.getProcessorId())),
                Map.entry("lastError", nullToEmpty(job.getLastError())),
                Map.entry("lockExpiresAt", job.getLockExpiresAt() == null ? "" : String.valueOf(toMillis(job.getLockExpiresAt()))),
                Map.entry("updatedAt", String.valueOf(toMillis(job.getUpdatedAt())))
        );
    }

    private String stateKey(String jobId) {
        return STATE_PREFIX + jobId;
    }

    private long toMillis(Instant instant) {
        return instant.toEpochMilli();
    }

    private String nullToEmpty(String value) {
        return value == null ? "" : value;
    }
}
