package com.example.jobqueue.service;

public class JobNotFoundException extends RuntimeException {

    public JobNotFoundException(String jobId) {
        super("Job %s was not found".formatted(jobId));
    }
}
