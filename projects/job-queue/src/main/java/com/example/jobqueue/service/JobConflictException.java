package com.example.jobqueue.service;

public class JobConflictException extends RuntimeException {

    public JobConflictException(String message) {
        super(message);
    }
}
