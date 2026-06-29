package com.example.jobqueue.service;

import java.time.Duration;

import com.example.jobqueue.config.JobQueueProperties;

import org.springframework.stereotype.Component;

@Component
public class RetryBackoffPolicy {

    private final JobQueueProperties properties;

    public RetryBackoffPolicy(JobQueueProperties properties) {
        this.properties = properties;
    }

    public Duration nextDelay(int retryCount) {
        long multiplier = 1L << Math.min(Math.max(retryCount - 1, 0), 10);
        Duration candidate = properties.getBaseRetryDelay().multipliedBy(multiplier);
        if (candidate.compareTo(properties.getMaxRetryDelay()) > 0) {
            return properties.getMaxRetryDelay();
        }
        return candidate;
    }
}
