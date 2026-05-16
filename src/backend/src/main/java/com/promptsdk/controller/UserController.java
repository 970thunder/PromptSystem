package com.promptsdk.controller;

import com.promptsdk.dto.*;
import com.promptsdk.service.UserService;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api/v1/user")
@RequiredArgsConstructor
public class UserController {

    private final UserService userService;

    @PostMapping("/login")
    public ApiResponse<LoginResponse> login(@Valid @RequestBody LoginRequest request) {
        return userService.login(request);
    }

    @PostMapping("/register")
    public ApiResponse<LoginResponse> register(@Valid @RequestBody RegisterRequest request) {
        return userService.register(request);
    }

    @GetMapping("/info")
    public ApiResponse<UserInfoVO> getUserInfo(@RequestHeader("Authorization") String token) {
        return userService.getUserInfo(token);
    }

    @PutMapping("/info")
    public ApiResponse<Void> updateUserInfo(
            @RequestHeader("Authorization") String token,
            @RequestBody UpdateUserRequest request) {
        return userService.updateUserInfo(token, request);
    }

    @PostMapping("/logout")
    public ApiResponse<Void> logout(@RequestHeader("Authorization") String token) {
        return userService.logout(token);
    }

    @PostMapping("/captcha")
    public ApiResponse<Void> sendCaptcha(@RequestParam String email) {
        return userService.sendCaptcha(email);
    }

}
