package com.promptsdk.mapper;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.promptsdk.entity.User;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Select;

@Mapper
public interface UserMapper extends BaseMapper<User> {

    @Select("SELECT * FROM users WHERE email = #{email} LIMIT 1")
    User selectByEmail(String email);
}
