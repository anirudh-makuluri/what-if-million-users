package com.example.jobqueue.web;

import jakarta.validation.constraints.Min;
import jakarta.validation.constraints.NotBlank;

public record ClaimJobRequest(
        @NotBlank(message = "processorId is required")
        String processorId,
        @Min(value = 1, message = "leaseSeconds must be at least 1")
        Long leaseSeconds
) {
}
