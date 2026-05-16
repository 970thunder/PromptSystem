package com.promptsdk.entity;

import com.baomidou.mybatisplus.annotation.*;
import lombok.Data;
import java.time.LocalDateTime;

@Data
@TableName("skills")
public class Skill {
    @TableId(type = IdType.AUTO)
    private Long id;

    private String name;

    private String description;

    private String cover;

    private String workflow;

    private String schema;

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
