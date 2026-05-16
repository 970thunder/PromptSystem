package com.promptsdk.service.impl;

import com.promptsdk.config.JwtConfig;
import com.promptsdk.dto.*;
import com.promptsdk.entity.User;
import com.promptsdk.mapper.UserMapper;
import com.promptsdk.service.UserService;
import lombok.RequiredArgsConstructor;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.stereotype.Service;

import java.util.concurrent.TimeUnit;

@Service
@RequiredArgsConstructor
public class UserServiceImpl implements UserService {

    private final UserMapper userMapper;
    private final JwtConfig jwtConfig;
    private final RedisTemplate<String, Object> redisTemplate;

    @Override
    public ApiResponse<LoginResponse> login(LoginRequest request) {
        User user = userMapper.selectByEmail(request.getEmail());
        if (user == null) {
            return ApiResponse.error(401, "Invalid email or password");
        }
        // Note: In production, use BCrypt to verify password
        if (!user.getPassword().equals(request.getPassword())) {
            return ApiResponse.error(401, "Invalid email or password");
        }

        String token = jwtConfig.generateToken(user.getId(), user.getEmail());

        LoginResponse response = new LoginResponse();
        response.setToken(token);
        response.setUser(convertToUserInfo(user));
        return ApiResponse.success(response);
    }

    @Override
    public ApiResponse<LoginResponse> register(RegisterRequest request) {
        // Validate captcha
        String cachedCaptcha = (String) redisTemplate.opsForValue().get("captcha:" + request.getEmail());
        if (cachedCaptcha == null || !cachedCaptcha.equals(request.getCaptcha())) {
            return ApiResponse.error(400, "Invalid or expired captcha");
        }

        // Check if email already exists
        User existingUser = userMapper.selectByEmail(request.getEmail());
        if (existingUser != null) {
            return ApiResponse.error(400, "Email already registered");
        }

        User user = new User();
        user.setUsername(request.getUsername());
        user.setEmail(request.getEmail());
        user.setPassword(request.getPassword()); // Note: In production, hash the password
        user.setLevel(1);
        user.setExperience(0);
        user.setStatus(1);

        userMapper.insert(user);

        String token = jwtConfig.generateToken(user.getId(), user.getEmail());

        LoginResponse response = new LoginResponse();
        response.setToken(token);
        response.setUser(convertToUserInfo(user));
        return ApiResponse.success(response);
    }

    @Override
    public ApiResponse<UserInfoVO> getUserInfo(String token) {
        if (token == null || !token.startsWith("Bearer ")) {
            return ApiResponse.error(401, "Invalid token");
        }
        token = token.substring(7);

        if (!jwtConfig.validateToken(token)) {
            return ApiResponse.error(401, "Invalid or expired token");
        }

        Long userId = jwtConfig.getUserIdFromToken(token);
        User user = userMapper.selectById(userId);
        if (user == null) {
            return ApiResponse.error(404, "User not found");
        }

        return ApiResponse.success(convertToUserInfo(user));
    }

    @Override
    public ApiResponse<Void> updateUserInfo(String token, UpdateUserRequest request) {
        if (token == null || !token.startsWith("Bearer ")) {
            return ApiResponse.error(401, "Invalid token");
        }
        token = token.substring(7);

        if (!jwtConfig.validateToken(token)) {
            return ApiResponse.error(401, "Invalid or expired token");
        }

        Long userId = jwtConfig.getUserIdFromToken(token);
        User user = userMapper.selectById(userId);
        if (user == null) {
            return ApiResponse.error(404, "User not found");
        }

        if (request.getUsername() != null) {
            user.setUsername(request.getUsername());
        }
        if (request.getAvatar() != null) {
            user.setAvatar(request.getAvatar());
        }
        if (request.getBio() != null) {
            user.setBio(request.getBio());
        }

        userMapper.updateById(user);
        return ApiResponse.success();
    }

    @Override
    public ApiResponse<Void> logout(String token) {
        if (token != null && token.startsWith("Bearer ")) {
            token = token.substring(7);
            // Add token to blacklist (simplified implementation)
            redisTemplate.opsForValue().set("blacklist:" + token, "1", 24, TimeUnit.HOURS);
        }
        return ApiResponse.success();
    }

    @Override
    public ApiResponse<Void> sendCaptcha(String email) {
        // Generate 6-digit captcha
        String captcha = String.valueOf((int) ((Math.random() * 900000) + 100000));
        // Store in Redis with 5-minute expiration
        redisTemplate.opsForValue().set("captcha:" + email, captcha, 5, TimeUnit.MINUTES);
        // Note: In production, send email here
        System.out.println("Captcha for " + email + ": " + captcha);
        return ApiResponse.success();
    }

    private UserInfoVO convertToUserInfo(User user) {
        UserInfoVO vo = new UserInfoVO();
        vo.setId(user.getId());
        vo.setUsername(user.getUsername());
        vo.setAvatar(user.getAvatar());
        vo.setEmail(user.getEmail());
        vo.setBio(user.getBio());
        vo.setLevel(user.getLevel());
        vo.setExperience(user.getExperience());
        return vo;
    }
}
