package com.example.jobqueue.web;

import com.example.jobqueue.service.JobService;

import jakarta.validation.Valid;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/jobs")
public class JobController {

    private final JobService jobService;

    public JobController(JobService jobService) {
        this.jobService = jobService;
    }

    @PostMapping
    @ResponseStatus(HttpStatus.CREATED)
    public JobResponse createJob(@Valid @RequestBody CreateJobRequest request) {
        return jobService.createJob(request);
    }

    @PostMapping("/{jobId}/claim")
    public JobResponse claimJob(@PathVariable String jobId, @Valid @RequestBody ClaimJobRequest request) {
        return jobService.claimJob(jobId, request);
    }

    @PostMapping("/{jobId}/complete")
    public JobResponse completeJob(@PathVariable String jobId, @Valid @RequestBody CompleteJobRequest request) {
        return jobService.completeJob(jobId, request);
    }

    @PostMapping("/{jobId}/fail")
    public JobResponse failJob(@PathVariable String jobId, @Valid @RequestBody FailJobRequest request) {
        return jobService.failJob(jobId, request);
    }

    @GetMapping("/{jobId}")
    public JobResponse getJob(@PathVariable String jobId) {
        return jobService.getJob(jobId);
    }
}
