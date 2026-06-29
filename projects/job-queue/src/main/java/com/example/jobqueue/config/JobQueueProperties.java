package com.example.jobqueue.config;

import java.time.Duration;

import org.springframework.boot.context.properties.ConfigurationProperties;

@ConfigurationProperties(prefix = "job-queue")
public class JobQueueProperties {

    private int defaultMaxRetries = 3;
    private Duration defaultLease = Duration.ofSeconds(60);
    private Duration baseRetryDelay = Duration.ofSeconds(15);
    private Duration maxRetryDelay = Duration.ofMinutes(10);
    private Duration hydrationLookAhead = Duration.ofMinutes(5);
    private int recoveryBatchSize = 100;
    private int hydrationBatchSize = 200;
    private long recoveryIntervalMs = 10_000L;
    private long hydrationIntervalMs = 15_000L;

    public int getDefaultMaxRetries() {
        return defaultMaxRetries;
    }

    public void setDefaultMaxRetries(int defaultMaxRetries) {
        this.defaultMaxRetries = defaultMaxRetries;
    }

    public Duration getDefaultLease() {
        return defaultLease;
    }

    public void setDefaultLease(Duration defaultLease) {
        this.defaultLease = defaultLease;
    }

    public Duration getBaseRetryDelay() {
        return baseRetryDelay;
    }

    public void setBaseRetryDelay(Duration baseRetryDelay) {
        this.baseRetryDelay = baseRetryDelay;
    }

    public Duration getMaxRetryDelay() {
        return maxRetryDelay;
    }

    public void setMaxRetryDelay(Duration maxRetryDelay) {
        this.maxRetryDelay = maxRetryDelay;
    }

    public Duration getHydrationLookAhead() {
        return hydrationLookAhead;
    }

    public void setHydrationLookAhead(Duration hydrationLookAhead) {
        this.hydrationLookAhead = hydrationLookAhead;
    }

    public int getRecoveryBatchSize() {
        return recoveryBatchSize;
    }

    public void setRecoveryBatchSize(int recoveryBatchSize) {
        this.recoveryBatchSize = recoveryBatchSize;
    }

    public int getHydrationBatchSize() {
        return hydrationBatchSize;
    }

    public void setHydrationBatchSize(int hydrationBatchSize) {
        this.hydrationBatchSize = hydrationBatchSize;
    }

    public long getRecoveryIntervalMs() {
        return recoveryIntervalMs;
    }

    public void setRecoveryIntervalMs(long recoveryIntervalMs) {
        this.recoveryIntervalMs = recoveryIntervalMs;
    }

    public long getHydrationIntervalMs() {
        return hydrationIntervalMs;
    }

    public void setHydrationIntervalMs(long hydrationIntervalMs) {
        this.hydrationIntervalMs = hydrationIntervalMs;
    }
}
