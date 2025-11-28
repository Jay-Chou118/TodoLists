package db

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// JWT密钥
var jwtSecret []byte

// 初始化JWT密钥
func init() {
	// 生成随机密钥（实际应用中应该从环境变量或配置文件中读取）
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		// 如果随机生成失败，使用默认密钥
		jwtSecret = []byte("default_secret_key_for_development")
	} else {
		jwtSecret = key
	}
}

// 自定义JWT声明结构
type Claims struct {
	UserID   string `json:"user_id"`
	DeviceID string `json:"device_id"`
	jwt.RegisteredClaims
}

// 生成密码哈希
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// 验证密码
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// 用户注册
func RegisterUser(username, password, email string) (*User, error) {
	// 检查用户名是否已存在
	for _, user := range Users {
		if user.Username == username {
			return nil, errors.New("用户名已存在")
		}
		if user.Email == email {
			return nil, errors.New("邮箱已被注册")
		}
	}

	// 生成密码哈希
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	// 创建新用户
	newUser := &User{
		ID:        generateUUID(),
		Username:  username,
		Password:  hashedPassword,
		Email:     email,
		CreatedAt: time.Now(),
	}

	// 添加到用户列表
	Users = append(Users, *newUser)

	return newUser, nil
}

// 用户登录
func LoginUser(username, password, deviceName, deviceID string) (*User, *Device, string, error) {
	// 查找用户
	var user *User
	for i, u := range Users {
		if u.Username == username {
			user = &Users[i]
			break
		}
	}

	if user == nil {
		return nil, nil, "", errors.New("用户名或密码错误")
	}

	// 验证密码
	if !CheckPassword(password, user.Password) {
		return nil, nil, "", errors.New("用户名或密码错误")
	}

	// 查找或创建设备
	var device *Device
	found := false
	for i, d := range Devices {
		if d.UserID == user.ID && d.DeviceID == deviceID {
			// 更新设备最后活跃时间
			Devices[i].LastSeen = time.Now()
			if deviceName != "" {
				Devices[i].Name = deviceName
			}
			device = &Devices[i]
			found = true
			break
		}
	}

	// 如果设备不存在，创建新设备
	if !found {
		if deviceName == "" {
			deviceName = "Unknown Device"
		}
		newDevice := Device{
			ID:        generateUUID(),
			UserID:    user.ID,
			Name:      deviceName,
			DeviceID:  deviceID,
			LastSeen:  time.Now(),
			CreatedAt: time.Now(),
		}
		Devices = append(Devices, newDevice)
		device = &Devices[len(Devices)-1]
	}

	// 生成JWT token
	token, err := generateToken(user.ID, device.DeviceID)
	if err != nil {
		return nil, nil, "", err
	}

	return user, device, token, nil
}

// 生成JWT token
func generateToken(userID, deviceID string) (string, error) {
	expireTime := time.Now().Add(24 * time.Hour) // 24小时过期

	claims := &Claims{
		UserID:   userID,
		DeviceID: deviceID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// 验证JWT token
func ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("无效的token")
}

// 生成UUID（简化版）
func generateUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return time.Now().Format("20060102150405")
	}
	return base64.URLEncoding.EncodeToString(b)[:22]
}

// 获取用户信息（不包含密码）
func GetUserByID(userID string) (*User, error) {
	for _, user := range Users {
		if user.ID == userID {
			// 创建副本并清空密码
			userCopy := user
			userCopy.Password = ""
			return &userCopy, nil
		}
	}
	return nil, errors.New("用户不存在")
}

// 获取用户的所有设备
func GetUserDevices(userID string) []Device {
	var devices []Device
	for _, device := range Devices {
		if device.UserID == userID {
			devices = append(devices, device)
		}
	}
	return devices
}
