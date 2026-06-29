package com.example.jobqueue.web;

import jakarta.validation.constraints.Min;
import jakarta.validation.constraints.NotBlank;

public record CreateJobRequest(
        @NotBlank(message = "payload is required")
        String payload,
        @Min(value = 0, message = "maxRetries must be zero or greater")
        Integer maxRetries,
        @Min(value = 0, message = "delaySeconds must be zero or greater")
        Long delaySeconds
) {
}
