package com.promptsdk.entity;

import com.baomidou.mybatisplus.annotation.*;
import lombok.Data;
import java.time.LocalDateTime;

@Data
@TableName("categories")
public class Category {
    @TableId(type = IdType.AUTO)
    private Long id;

    private String name;

    private String icon;

    private Integer sort;

    private Integer type;

    @TableField(fill = FieldFill.INSERT)
    private LocalDateTime createdAt;
}
