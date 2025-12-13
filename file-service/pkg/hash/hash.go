package hash

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
)

type HashAlgorithm string

const (
	MD5    HashAlgorithm = "md5"
	SHA1   HashAlgorithm = "sha1"
	SHA256 HashAlgorithm = "sha256"
	SHA512 HashAlgorithm = "sha512"
)

type FileHashResult struct {
	Algorithm HashAlgorithm
	Hash      string
	FileSize  int64
	FileName  string
}

type Hasher interface {
	Calculate(data []byte) (string, error)
	CalculateFile(filePath string) (*FileHashResult, error)
	CalculateReader(reader io.Reader) (string, error)
	Verify(data []byte, expectedHash string) (bool, error)
}

type FileHasher struct {
	algorithm HashAlgorithm
}

func NewFileHasher(algorithm HashAlgorithm) *FileHasher {
	return &FileHasher{
		algorithm: algorithm,
	}
}

func (h *FileHasher) Calculate(data []byte) (string, error) {
	hasher, err := h.getHasher()
	if err != nil {
		return "", err
	}

	hasher.Write(data)
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (h *FileHasher) CalculateFile(filePath string) (*FileHashResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file stats: %w", err)
	}

	hashStr, err := h.CalculateReader(file)
	if err != nil {
		return nil, err
	}

	return &FileHashResult{
		Algorithm: h.algorithm,
		Hash:      hashStr,
		FileSize:  stat.Size(),
		FileName:  stat.Name(),
	}, nil
}

func (h *FileHasher) CalculateReader(reader io.Reader) (string, error) {
	hasher, err := h.getHasher()
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(hasher, reader); err != nil {
		return "", fmt.Errorf("failed to read data: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (h *FileHasher) Verify(data []byte, expectedHash string) (bool, error) {
	calculatedHash, err := h.Calculate(data)
	if err != nil {
		return false, err
	}

	return calculatedHash == expectedHash, nil
}

func (h *FileHasher) getHasher() (hash.Hash, error) {
	switch h.algorithm {
	case MD5:
		return md5.New(), nil
	case SHA1:
		return sha1.New(), nil
	case SHA256:
		return sha256.New(), nil
	case SHA512:
		return sha512.New(), nil
	default:
		return nil, fmt.Errorf("unsupported hash algorithm: %s", h.algorithm)
	}
}

func CompareFiles(file1, file2 string, algorithm HashAlgorithm) (bool, error) {
	hasher := NewFileHasher(algorithm)

	hash1, err := hasher.CalculateFile(file1)
	if err != nil {
		return false, err
	}

	hash2, err := hasher.CalculateFile(file2)
	if err != nil {
		return false, err
	}

	return hash1.Hash == hash2.Hash, nil
}
