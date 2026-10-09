package service

import (
	"errors"
	"fmt"
	"time"

	"sshtunnelhub/internal/crypto"
	"sshtunnelhub/internal/db"
	"sshtunnelhub/internal/model"
	"sshtunnelhub/internal/sshutil"
)

type HostService struct{}

func NewHostService() *HostService {
	return &HostService{}
}

type CreateHostReq struct {
	Name       string `json:"name" binding:"required"`
	Host       string `json:"host" binding:"required"`
	Port       int    `json:"port"`
	Username   string `json:"username" binding:"required"`
	AuthType   string `json:"auth_type" binding:"required"` // "password" | "private_key"
	Password   string `json:"password"`
	PrivateKey string `json:"private_key"`
	Passphrase string `json:"passphrase"`
	Remark     string `json:"remark"`
}

type UpdateHostReq struct {
	Name       string `json:"name" binding:"required"`
	Host       string `json:"host" binding:"required"`
	Port       int    `json:"port"`
	Username   string `json:"username" binding:"required"`
	AuthType   string `json:"auth_type" binding:"required"`
	Password   string `json:"password"`    // If empty, keep existing
	PrivateKey string `json:"private_key"`// If empty, keep existing
	Passphrase string `json:"passphrase"` // If empty, keep existing
	Remark     string `json:"remark"`
}

type TestHostReq struct {
	Host       string `json:"host" binding:"required"`
	Port       int    `json:"port"`
	Username   string `json:"username" binding:"required"`
	AuthType   string `json:"auth_type" binding:"required"`
	Password   string `json:"password"`
	PrivateKey string `json:"private_key"`
	Passphrase string `json:"passphrase"`
}

func (s *HostService) ListHosts() ([]model.HostVO, error) {
	var hosts []model.Host
	if err := db.DB.Order("id desc").Find(&hosts).Error; err != nil {
		return nil, err
	}

	res := make([]model.HostVO, len(hosts))
	for i, h := range hosts {
		res[i] = h.ToVO()
	}
	return res, nil
}

func (s *HostService) GetHost(id uint) (*model.HostVO, error) {
	var h model.Host
	if err := db.DB.First(&h, id).Error; err != nil {
		return nil, err
	}
	vo := h.ToVO()
	return &vo, nil
}

func (s *HostService) CreateHost(req CreateHostReq) (*model.HostVO, error) {
	if req.Port <= 0 {
		req.Port = 22
	}

	encPass, err := crypto.Encrypt(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt password: %w", err)
	}
	encKey, err := crypto.Encrypt(req.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt private key: %w", err)
	}
	encPassphrase, err := crypto.Encrypt(req.Passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt passphrase: %w", err)
	}

	h := model.Host{
		Name:       req.Name,
		Host:       req.Host,
		Port:       req.Port,
		Username:   req.Username,
		AuthType:   req.AuthType,
		Password:   encPass,
		PrivateKey: encKey,
		Passphrase: encPassphrase,
		Remark:     req.Remark,
	}

	if err := db.DB.Create(&h).Error; err != nil {
		return nil, err
	}

	vo := h.ToVO()
	return &vo, nil
}

func (s *HostService) UpdateHost(id uint, req UpdateHostReq) (*model.HostVO, error) {
	var h model.Host
	if err := db.DB.First(&h, id).Error; err != nil {
		return nil, err
	}

	if req.Port <= 0 {
		req.Port = 22
	}

	h.Name = req.Name
	h.Host = req.Host
	h.Port = req.Port
	h.Username = req.Username
	h.AuthType = req.AuthType
	h.Remark = req.Remark

	// Only update encrypted secrets if provided
	if req.Password != "" {
		encPass, err := crypto.Encrypt(req.Password)
		if err != nil {
			return nil, err
		}
		h.Password = encPass
	}
	if req.PrivateKey != "" {
		encKey, err := crypto.Encrypt(req.PrivateKey)
		if err != nil {
			return nil, err
		}
		h.PrivateKey = encKey
	}
	if req.Passphrase != "" {
		encPassphrase, err := crypto.Encrypt(req.Passphrase)
		if err != nil {
			return nil, err
		}
		h.Passphrase = encPassphrase
	}

	if err := db.DB.Save(&h).Error; err != nil {
		return nil, err
	}

	vo := h.ToVO()
	return &vo, nil
}

func (s *HostService) DeleteHost(id uint) error {
	// Check if any tunnels rely on this host
	var count int64
	if err := db.DB.Model(&model.Tunnel{}).Where("host_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("cannot delete host because it has active tunnels attached")
	}

	return db.DB.Delete(&model.Host{}, id).Error
}

func (s *HostService) TestSavedHost(id uint) (int64, string, error) {
	var h model.Host
	if err := db.DB.First(&h, id).Error; err != nil {
		return 0, "", err
	}
	latency, banner, err := sshutil.TestConnection(&h)
	return latency.Milliseconds(), banner, err
}

func (s *HostService) TestRawHost(req TestHostReq) (int64, string, error) {
	if req.Port <= 0 {
		req.Port = 22
	}

	encPass, _ := crypto.Encrypt(req.Password)
	encKey, _ := crypto.Encrypt(req.PrivateKey)
	encPassphrase, _ := crypto.Encrypt(req.Passphrase)

	tempHost := model.Host{
		Host:       req.Host,
		Port:       req.Port,
		Username:   req.Username,
		AuthType:   req.AuthType,
		Password:   encPass,
		PrivateKey: encKey,
		Passphrase: encPassphrase,
	}

	start := time.Now()
	latency, banner, err := sshutil.TestConnection(&tempHost)
	if err != nil {
		return time.Since(start).Milliseconds(), "", err
	}
	return latency.Milliseconds(), banner, nil
}
