package com.promptsdk.entity;

import com.baomidou.mybatisplus.annotation.*;
import lombok.Data;
import java.time.LocalDateTime;

@Data
@TableName("prompts")
public class Prompt {
    @TableId(type = IdType.AUTO)
    private Long id;

    private String title;

    private String description;

    private String cover;

    private String content;

    private String systemPrompt;

    private String model;

    private String params;

    private Long categoryId;

    private Long userId;

    private Integer views;

    private Integer likes;

    private Integer favorites;

    private Integer status;

    @TableField(fill = FieldFill.INSERT)
    private LocalDateTime createdAt;

    @TableField(fill = FieldFill.INSERT_UPDATE)
    private LocalDateTime updatedAt;
}
