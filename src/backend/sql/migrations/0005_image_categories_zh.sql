-- Add / refresh Chinese image-focused prompt categories (IDs 1-30).
USE promptos;

INSERT INTO categories (id, name, icon, sort, type) VALUES
(1, '摄影', 'camera', 1, 1),
(2, '插画', 'brush', 2, 1),
(3, '3D', 'cube', 3, 1),
(4, '电商', 'shopping', 4, 1),
(5, '人像', 'portrait', 5, 1),
(6, '建筑', 'building', 6, 1),
(7, '动漫', 'anime', 7, 1),
(8, 'UI', 'layout', 8, 1),
(9, '海报', 'poster', 9, 1),
(10, '产品', 'product', 10, 1),
(11, '风景', 'landscape', 11, 1),
(12, '美食', 'food', 12, 1),
(13, '时尚', 'fashion', 13, 1),
(14, '游戏', 'game', 14, 1),
(15, '图标', 'icon', 15, 1),
(16, 'LOGO', 'logo', 16, 1),
(17, '室内设计', 'interior', 17, 1),
(18, '汽车', 'car', 18, 1),
(19, '宠物', 'pet', 19, 1),
(20, '婚礼', 'wedding', 20, 1),
(21, '科幻', 'scifi', 21, 1),
(22, '水彩', 'watercolor', 22, 1),
(23, '油画', 'oil', 23, 1),
(24, '像素', 'pixel', 24, 1),
(25, '线稿', 'lineart', 25, 1),
(26, '表情包', 'emoji', 26, 1),
(27, '壁纸', 'wallpaper', 27, 1),
(28, '社交媒体', 'social', 28, 1),
(29, '广告创意', 'ads', 29, 1),
(30, '其他', 'more', 30, 1)
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  icon = VALUES(icon),
  sort = VALUES(sort),
  type = VALUES(type);
