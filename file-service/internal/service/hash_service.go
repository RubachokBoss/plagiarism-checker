package service

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"strings"
)

type HashService interface {
	CalculateHash(data []byte) (string, error)
	CalculateHashFromString(data string) (string, error)
	VerifyHash(data []byte, expectedHash string) (bool, error)
	GetHashAlgorithm() string
}

type hashService struct {
	algorithm string
}

func NewHashService(algorithm string) HashService {
	return &hashService{
		algorithm: strings.ToLower(algorithm),
	}
}

func (s *hashService) CalculateHash(data []byte) (string, error) {
	hasher, err := s.getHasher()
	if err != nil {
		return "", err
	}

	hasher.Write(data)
	hashBytes := hasher.Sum(nil)
	return hex.EncodeToString(hashBytes), nil
}

func (s *hashService) CalculateHashFromString(data string) (string, error) {
	return s.CalculateHash([]byte(data))
}

func (s *hashService) VerifyHash(data []byte, expectedHash string) (bool, error) {
	calculatedHash, err := s.CalculateHash(data)
	if err != nil {
		return false, err
	}

	return strings.EqualFold(calculatedHash, expectedHash), nil
}

func (s *hashService) GetHashAlgorithm() string {
	return s.algorithm
}

func (s *hashService) getHasher() (hash.Hash, error) {
	switch s.algorithm {
	case "md5":
		return md5.New(), nil
	case "sha1":
		return sha1.New(), nil
	case "sha256":
		return sha256.New(), nil
	case "sha512":
		return sha512.New(), nil
	default:
		return nil, fmt.Errorf("unsupported hash algorithm: %s", s.algorithm)
	}
}

type FileHashCalculator struct {
	hashService HashService
}

func NewFileHashCalculator(algorithm string) *FileHashCalculator {
	return &FileHashCalculator{
		hashService: NewHashService(algorithm),
	}
}

func (c *FileHashCalculator) CalculateFileHash(data []byte) (string, error) {
	return c.hashService.CalculateHash(data)
}

func (c *FileHashCalculator) GetAlgorithm() string {
	return c.hashService.GetHashAlgorithm()
}

func (c *FileHashCalculator) CompareFiles(file1, file2 []byte) (bool, error) {
	hash1, err := c.CalculateFileHash(file1)
	if err != nil {
		return false, err
	}

	hash2, err := c.CalculateFileHash(file2)
	if err != nil {
		return false, err
	}

	return hash1 == hash2, nil
}
