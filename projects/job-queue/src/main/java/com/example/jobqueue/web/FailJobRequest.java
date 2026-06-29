package com.example.jobqueue.web;

import jakarta.validation.constraints.NotBlank;

public record FailJobRequest(
        @NotBlank(message = "processorId is required")
        String processorId,
        @NotBlank(message = "errorMessage is required")
        String errorMessage,
        Boolean retryable
) {
}
