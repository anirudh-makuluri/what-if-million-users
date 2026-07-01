package com.example.leaderboard.config;

import org.springframework.boot.context.properties.ConfigurationProperties;

@ConfigurationProperties(prefix = "leaderboard")
public class LeaderboardProperties {

    private int hydrationBatchSize = 500;
    private long cacheAuditIntervalMs = 30_000L;
    private int maxWriteRetries = 5;

    public int getHydrationBatchSize() {
        return hydrationBatchSize;
    }

    public void setHydrationBatchSize(int hydrationBatchSize) {
        this.hydrationBatchSize = hydrationBatchSize;
    }

    public long getCacheAuditIntervalMs() {
        return cacheAuditIntervalMs;
    }

    public void setCacheAuditIntervalMs(long cacheAuditIntervalMs) {
        this.cacheAuditIntervalMs = cacheAuditIntervalMs;
    }

    public int getMaxWriteRetries() {
        return maxWriteRetries;
    }

    public void setMaxWriteRetries(int maxWriteRetries) {
        this.maxWriteRetries = maxWriteRetries;
    }
}
