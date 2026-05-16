package com.promptsdk.dto;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import lombok.Data;
import java.util.List;

@Data
public class PublishPromptRequest {
    @NotBlank(message = "Title cannot be blank")
    private String title;

    private String description;

    private String cover;

    @NotBlank(message = "Content cannot be blank")
    private String content;

    private String systemPrompt;

    private String model;

    private String params;

    @NotNull(message = "Category ID cannot be null")
    private Long categoryId;

    private List<String> tags;
}
