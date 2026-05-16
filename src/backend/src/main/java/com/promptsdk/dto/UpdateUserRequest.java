package com.promptsdk.dto;

import lombok.Data;

@Data
public class UpdateUserRequest {
    private String username;
    private String avatar;
    private String bio;
}
