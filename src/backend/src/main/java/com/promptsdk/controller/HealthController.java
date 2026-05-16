package com.promptsdk.controller;

import com.promptsdk.dto.ApiResponse;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.HashMap;
import java.util.Map;

@RestController
@RequestMapping("/api/v1/health")
public class HealthController {

    @GetMapping
    public ApiResponse<Map<String, String>> health() {
        Map<String, String> data = new HashMap<>();
        data.put("status", "ok");
        data.put("service", "promptos-backend");
        return ApiResponse.success(data);
    }
}
