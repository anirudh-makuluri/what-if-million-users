package com.example.jobqueue.web;

import jakarta.validation.constraints.NotBlank;

public record CompleteJobRequest(
        @NotBlank(message = "processorId is required")
        String processorId
) {
}
