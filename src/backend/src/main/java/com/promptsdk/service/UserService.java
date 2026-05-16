package com.promptsdk.service;

import com.promptsdk.dto.*;

public interface UserService {
    ApiResponse<LoginResponse> login(LoginRequest request);
    ApiResponse<LoginResponse> register(RegisterRequest request);
    ApiResponse<UserInfoVO> getUserInfo(String token);
    ApiResponse<Void> updateUserInfo(String token, UpdateUserRequest request);
    ApiResponse<Void> logout(String token);
    ApiResponse<Void> sendCaptcha(String email);
}
