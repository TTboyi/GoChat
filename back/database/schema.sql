from pathlib import Path

schema_sql = """
-- 用户信息表
CREATE TABLE user_info (
    id BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '自增id',
    uuid CHAR(8) UNIQUE NOT NULL COMMENT '用户唯一id',
    nickname VARCHAR(20) NOT NULL COMMENT '昵称',
    telephone CHAR(11) NOT NULL COMMENT '电话',
    email CHAR(30) COMMENT '邮箱',
    avatar CHAR(255) NOT NULL DEFAULT 'https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png' COMMENT '头像',
    gender TINYINT COMMENT '性别，0.男，1.女',
    signature VARCHAR(100) COMMENT '个性签名',
    password CHAR(60) NOT NULL COMMENT '密码hash',
    birthday DATE COMMENT '生日',
    is_admin TINYINT NOT NULL DEFAULT 0 COMMENT '是否是管理员，0.否，1.是',
    status TINYINT NOT NULL DEFAULT 0 COMMENT '状态，0.正常，1.禁用',
    created_at DATETIME NOT NULL COMMENT '创建时间',
    deleted_at DATETIME COMMENT '删除时间',
    INDEX idx_telephone (telephone),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户信息表';

-- 群组信息表
CREATE TABLE group_info (
    id BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '自增id',
    uuid CHAR(6) UNIQUE NOT NULL COMMENT '群组唯一id',
    name VARCHAR(20) NOT NULL COMMENT '群名称',
    notice VARCHAR(500) COMMENT '群公告',
    members JSON COMMENT '群组成员（冗余）',
    member_cnt INT DEFAULT 1 COMMENT '群人数',
    owner_id CHAR(20) NOT NULL COMMENT '群主uuid',
    add_mode TINYINT DEFAULT 0 COMMENT '加群方式，0.直接，1.审核',
    avatar CHAR(255) NOT NULL DEFAULT 'https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png' COMMENT '头像',
    status TINYINT DEFAULT 0 COMMENT '状态，0.正常，1.禁用，2.解散',
    created_at DATETIME NOT NULL COMMENT '创建时间',
    deleted_at DATETIME COMMENT '删除时间',
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='群组信息表';

-- 联系人表
CREATE TABLE user_contact (
    id BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '自增id',
    user_id CHAR(20) NOT NULL COMMENT '用户id',
    contact_id CHAR(20) NOT NULL COMMENT '联系对象id',
    contact_type TINYINT NOT NULL COMMENT '类型，0.用户，1.群聊',
    status TINYINT NOT NULL COMMENT '状态，0.正常，1.拉黑，2.被拉黑，3.删除，4.被删除，5.被禁言，6.退群，7.被踢',
    created_at DATETIME NOT NULL COMMENT '创建时间',
    deleted_at DATETIME COMMENT '删除时间',
    INDEX idx_user_contact (user_id, contact_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户联系人表';

-- 添加申请表
CREATE TABLE contact_apply (
    id BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '自增id',
    uuid CHAR(20) UNIQUE COMMENT '申请id',
    user_id CHAR(20) NOT NULL COMMENT '申请人id',
    contact_id CHAR(20) NOT NULL COMMENT '被申请id',
    contact_type TINYINT NOT NULL COMMENT '类型，0.用户，1.群聊',
    status TINYINT NOT NULL COMMENT '状态，0.申请中，1.通过，2.拒绝，3.拉黑',
    message VARCHAR(100) COMMENT '申请信息',
    last_apply_at DATETIME NOT NULL COMMENT '最后申请时间',
    deleted_at DATETIME COMMENT '删除时间',
    INDEX idx_user_contact_type (user_id, contact_id, contact_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='添加申请记录';

-- 会话表
CREATE TABLE session (
    id BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '自增id',
    uuid CHAR(20) UNIQUE COMMENT '会话uuid',
    send_id CHAR(20) NOT NULL COMMENT '创建者uuid',
    receive_id CHAR(20) NOT NULL COMMENT '接收者uuid',
    receive_name VARCHAR(20) NOT NULL COMMENT '会话对象名称',
    avatar CHAR(255) NOT NULL DEFAULT 'default_avatar.png' COMMENT '头像',
    created_at DATETIME COMMENT '创建时间',
    deleted_at DATETIME COMMENT '删除时间',
    UNIQUE KEY uniq_session (send_id, receive_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户会话表';

-- 消息表
CREATE TABLE message (
    id BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '自增id',
    uuid CHAR(20) UNIQUE NOT NULL COMMENT '消息uuid',
    session_id CHAR(20) NOT NULL COMMENT '会话uuid',
    type TINYINT NOT NULL COMMENT '类型，0.文本，1.文件，2.通话',
    content TEXT COMMENT '消息内容',
    url CHAR(255) COMMENT '消息url',
    send_id CHAR(20) NOT NULL COMMENT '发送者uuid',
    send_name VARCHAR(20) NOT NULL COMMENT '发送者昵称',
    send_avatar VARCHAR(255) NOT NULL COMMENT '发送者头像',
    receive_id CHAR(20) NOT NULL COMMENT '接收者uuid',
    file_type CHAR(10) COMMENT '文件类型',
    file_name VARCHAR(50) COMMENT '文件名',
    file_size CHAR(20) COMMENT '文件大小',
    status TINYINT NOT NULL DEFAULT 0 COMMENT '状态，0.未发送，1.已发送',
    av_data TEXT COMMENT '通话数据',
    created_at DATETIME NOT NULL COMMENT '创建时间',
    INDEX idx_session_id (session_id),
    INDEX idx_send_id (send_id),
    INDEX idx_receive_id (receive_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='消息记录表';
"""

output_dir = Path("chatapp/back/database")
output_dir.mkdir(parents=True, exist_ok=True)
file_path = output_dir / "schema.sql"
file_path.write_text(schema_sql.strip(), encoding="utf-8")

file_path.name
