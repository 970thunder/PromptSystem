package com.promptsdk.dto;

import lombok.Data;

@Data
public class UserInfoVO {
    private Long id;
    private String username;
    private String avatar;
    private String email;
    private String bio;
    private Integer level;
    private Integer experience;
}
