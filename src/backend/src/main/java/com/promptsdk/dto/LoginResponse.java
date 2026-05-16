package com.promptsdk.dto;

import lombok.Data;

@Data
public class LoginResponse {
    private String token;
    private UserInfoVO user;
}
