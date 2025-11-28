package db

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"time"
)

// 设备类型常量
const (
	DeviceTypePC      = "pc"
	DeviceTypeMobile  = "mobile"
	DeviceTypeTablet  = "tablet"
	DeviceTypeUnknown = "unknown"
)

// 设备操作系统常量
const (
	OSWindows = "windows"
	OSMacOS   = "macos"
	OSLinux   = "linux"
	OSAndroid = "android"
	OSiOS     = "ios"
	OSUnknown = "unknown"
)

// 设备浏览器常量
const (
	BrowserChrome  = "chrome"
	BrowserFirefox = "firefox"
	BrowserSafari  = "safari"
	BrowserEdge    = "edge"
	BrowserIE      = "ie"
	BrowserUnknown = "unknown"
)

// DeviceInfo 设备详细信息
type DeviceInfo struct {
	Type      string `json:"type"`       // 设备类型
	OS        string `json:"os"`         // 操作系统
	OSVersion string `json:"os_version"` // 操作系统版本
	Browser   string `json:"browser"`    // 浏览器
	Model     string `json:"model"`      // 设备型号
	AppVersion string `json:"app_version"` // 应用版本（如果是移动应用）
}

// 生成设备唯一标识符
func GenerateDeviceID(deviceName, userAgent, ip string) string {
	// 使用设备名称、用户代理和IP地址生成唯一标识
	hash := md5.Sum([]byte(deviceName + userAgent + ip + time.Now().Format(time.RFC3339))) // 添加时间戳以避免冲突
	return hex.EncodeToString(hash[:])
}

// 解析用户代理字符串获取设备信息
func ParseUserAgent(userAgent string) DeviceInfo {
	info := DeviceInfo{
		Type:      DeviceTypeUnknown,
		OS:        OSUnknown,
		OSVersion: "",
		Browser:   BrowserUnknown,
		Model:     "",
	}

	// 这里可以添加更详细的用户代理解析逻辑
	// 简化版本：
	if contains(userAgent, "Windows") {
		info.OS = OSWindows
	} else if contains(userAgent, "Macintosh") {
		info.OS = OSMacOS
	} else if contains(userAgent, "Linux") && !contains(userAgent, "Android") {
		info.OS = OSLinux
	} else if contains(userAgent, "Android") {
		info.OS = OSAndroid
		info.Type = DeviceTypeMobile
	} else if contains(userAgent, "iPhone") || contains(userAgent, "iPad") || contains(userAgent, "iPod") {
		info.OS = OSiOS
		if contains(userAgent, "iPad") {
			info.Type = DeviceTypeTablet
		} else {
			info.Type = DeviceTypeMobile
		}
	}

	if contains(userAgent, "Chrome") && !contains(userAgent, "Edg/") {
		info.Browser = BrowserChrome
	} else if contains(userAgent, "Firefox") {
		info.Browser = BrowserFirefox
	} else if contains(userAgent, "Safari") && !contains(userAgent, "Chrome") {
		info.Browser = BrowserSafari
	} else if contains(userAgent, "Edg/") || contains(userAgent, "Edge/") {
		info.Browser = BrowserEdge
	} else if contains(userAgent, "MSIE") || contains(userAgent, "Trident/") {
		info.Browser = BrowserIE
	}

	// 确定设备类型
	if info.Type == DeviceTypeUnknown {
		if contains(userAgent, "Mobile") || contains(userAgent, "Android") || contains(userAgent, "iPhone") {
			info.Type = DeviceTypeMobile
		} else if contains(userAgent, "iPad") || contains(userAgent, "Tablet") {
			info.Type = DeviceTypeTablet
		} else {
			info.Type = DeviceTypePC
		}
	}

	return info
}

// 根据用户ID和设备ID查找设备
func GetDeviceByUserAndDeviceID(userID, deviceID string) (*Device, error) {
	for i, device := range Devices {
		if device.UserID == userID && device.DeviceID == deviceID {
			return &Devices[i], nil
		}
	}
	return nil, errors.New("设备不存在")
}

// 创建设备记录
func CreateDevice(userID, deviceName, deviceID string) (*Device, error) {
	// 检查设备是否已存在
	existingDevice, _ := GetDeviceByUserAndDeviceID(userID, deviceID)
	if existingDevice != nil {
		return existingDevice, nil // 返回已存在的设备
	}

	newDevice := Device{
		ID:        generateUUID(),
		UserID:    userID,
		Name:      deviceName,
		DeviceID:  deviceID,
		LastSeen:  time.Now(),
		CreatedAt: time.Now(),
	}

	Devices = append(Devices, newDevice)
	return &newDevice, nil
}

// 更新设备最后活跃时间
func UpdateDeviceLastSeen(deviceID string) error {
	for i := range Devices {
		if Devices[i].DeviceID == deviceID {
			Devices[i].LastSeen = time.Now()
			return nil
		}
	}
	return errors.New("设备不存在")
}

// 更新设备名称
func UpdateDeviceName(userID, deviceID, newName string) error {
	for i := range Devices {
		if Devices[i].UserID == userID && Devices[i].DeviceID == deviceID {
			Devices[i].Name = newName
			return nil
		}
	}
	return errors.New("设备不存在或无权修改")
}

// 删除设备
func DeleteDevice(userID, deviceID string) error {
	for i, device := range Devices {
		if device.UserID == userID && device.DeviceID == deviceID {
			// 从列表中删除设备
			Devices = append(Devices[:i], Devices[i+1:]...)
			return nil
		}
	}
	return errors.New("设备不存在或无权删除")
}

// 获取最近活跃的设备（限制数量）
func GetRecentActiveDevices(userID string, limit int) []Device {
	var userDevices []Device

	// 收集用户的所有设备
	for _, device := range Devices {
		if device.UserID == userID {
			userDevices = append(userDevices, device)
		}
	}

	// 按最后活跃时间排序（简化版本，实际应该使用更高效的排序算法）
	for i := 0; i < len(userDevices); i++ {
		for j := i + 1; j < len(userDevices); j++ {
			if userDevices[i].LastSeen.Before(userDevices[j].LastSeen) {
				userDevices[i], userDevices[j] = userDevices[j], userDevices[i]
			}
		}
	}

	// 限制返回数量
	if len(userDevices) > limit {
		return userDevices[:limit]
	}

	return userDevices
}

// 检查设备是否已被授权（未被删除）
func IsDeviceAuthorized(userID, deviceID string) bool {
	for _, device := range Devices {
		if device.UserID == userID && device.DeviceID == deviceID {
			return true
		}
	}
	return false
}

// 辅助函数：检查字符串是否包含子字符串
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
