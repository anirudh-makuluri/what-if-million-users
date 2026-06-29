package com.example.jobqueue.web;

import java.time.Instant;

import com.example.jobqueue.model.JobStatus;

public record JobResponse(
        String id,
        String payload,
        JobStatus status,
        int retryCount,
        int maxRetries,
        String processorId,
        String lastError,
        Instant availableAt,
        Instant lockExpiresAt,
        Instant createdAt,
        Instant updatedAt,
        Instant completedAt
) {
}
