package hashing

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
)

// Hasher provides hashing utilities for feature flags
type Hasher struct{}

// NewHasher creates a new hasher instance
func NewHasher() *Hasher {
	return &Hasher{}
}

// HashUserKey hashes a user key for privacy and consistency
func (h *Hasher) HashUserKey(userKey string) string {
	hash := sha256.Sum256([]byte(userKey))
	return hex.EncodeToString(hash[:])
}

// GenerateBucketingID creates a deterministic bucketing ID for consistent assignment
// Formula: SHA256(env_salt + flag_key + user_key)
func (h *Hasher) GenerateBucketingID(envSalt, flagKey, userKey string) string {
	input := envSalt + flagKey + userKey
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// DeterministicBucket converts a bucketing ID to a bucket number (0-9999)
// This provides 10,000 buckets for fine-grained percentage rollouts
func (h *Hasher) DeterministicBucket(bucketingID string) int {
	// Take first 8 characters of hex string for consistency
	if len(bucketingID) < 8 {
		bucketingID = bucketingID + "00000000"[:8-len(bucketingID)]
	}

	hexStr := bucketingID[:8]

	// Convert hex to uint64
	val, err := strconv.ParseUint(hexStr, 16, 64)
	if err != nil {
		// Fallback: use first 4 bytes as uint32
		if len(bucketingID) >= 4 {
			val = uint64(bucketingID[0])<<24 |
				uint64(bucketingID[1])<<16 |
				uint64(bucketingID[2])<<8 |
				uint64(bucketingID[3])
		} else {
			val = 0
		}
	}

	// Map to 0-9999 range
	return int(val % 10000)
}

// IsInPercentageRange checks if a bucket falls within a percentage range
func (h *Hasher) IsInPercentageRange(bucket int, percentage float64) bool {
	if percentage <= 0 {
		return false
	}
	if percentage >= 100 {
		return true
	}

	// Convert percentage to bucket threshold (0-9999)
	threshold := int(percentage * 100) // percentage * 100 = bucket threshold
	return bucket < threshold
}

// IsInBucketRange checks if a bucket falls within a specific bucket range
func (h *Hasher) IsInBucketRange(bucket, start, end int) bool {
	return bucket >= start && bucket < end
}

// AllocateBucketsForVariations allocates bucket ranges for variations based on weights
func (h *Hasher) AllocateBucketsForVariations(weights []float64) []BucketRange {
	if len(weights) == 0 {
		return nil
	}

	// Normalize weights to percentages
	total := 0.0
	for _, weight := range weights {
		total += weight
	}

	if total == 0 {
		return nil
	}

	ranges := make([]BucketRange, len(weights))
	currentBucket := 0

	for i, weight := range weights {
		percentage := (weight / total) * 100.0
		bucketCount := int(percentage * 100) // Convert to bucket count (out of 10000)

		// Handle rounding for the last variation
		if i == len(weights)-1 {
			bucketCount = 10000 - currentBucket
		}

		ranges[i] = BucketRange{
			Start:      currentBucket,
			End:        currentBucket + bucketCount,
			Percentage: percentage,
		}

		currentBucket += bucketCount
	}

	return ranges
}

// BucketRange represents a range of buckets allocated to a variation
type BucketRange struct {
	Start      int     `json:"start"`
	End        int     `json:"end"`
	Percentage float64 `json:"percentage"`
}

// Contains checks if a bucket is within this range
func (br BucketRange) Contains(bucket int) bool {
	return bucket >= br.Start && bucket < br.End
}

// Size returns the number of buckets in this range
func (br BucketRange) Size() int {
	return br.End - br.Start
}

// BucketingResult holds the result of user bucketing
type BucketingResult struct {
	BucketingID string
	Bucket      int
	Variation   string
	Reason      string
}

// BucketUser performs complete user bucketing for a flag
func (h *Hasher) BucketUser(envSalt, flagKey, userKey string, variations []VariationAllocation) *BucketingResult {
	bucketingID := h.GenerateBucketingID(envSalt, flagKey, userKey)
	bucket := h.DeterministicBucket(bucketingID)

	// Find which variation this bucket belongs to
	for _, variation := range variations {
		if variation.BucketRange.Contains(bucket) {
			return &BucketingResult{
				BucketingID: bucketingID,
				Bucket:      bucket,
				Variation:   variation.Key,
				Reason:      fmt.Sprintf("bucketed into range %d-%d", variation.BucketRange.Start, variation.BucketRange.End),
			}
		}
	}

	// Fallback to first variation if no match (shouldn't happen with proper allocation)
	if len(variations) > 0 {
		return &BucketingResult{
			BucketingID: bucketingID,
			Bucket:      bucket,
			Variation:   variations[0].Key,
			Reason:      "fallback to default variation",
		}
	}

	return &BucketingResult{
		BucketingID: bucketingID,
		Bucket:      bucket,
		Variation:   "",
		Reason:      "no variations configured",
	}
}

// VariationAllocation represents how buckets are allocated to a variation
type VariationAllocation struct {
	Key         string      `json:"key"`
	Weight      float64     `json:"weight"`
	BucketRange BucketRange `json:"bucket_range"`
}

// CreateVariationAllocations creates bucket allocations for variations with given weights
func (h *Hasher) CreateVariationAllocations(variationKeys []string, weights []float64) []VariationAllocation {
	if len(variationKeys) != len(weights) {
		return nil
	}

	bucketRanges := h.AllocateBucketsForVariations(weights)
	if len(bucketRanges) != len(variationKeys) {
		return nil
	}

	allocations := make([]VariationAllocation, len(variationKeys))
	for i, key := range variationKeys {
		allocations[i] = VariationAllocation{
			Key:         key,
			Weight:      weights[i],
			BucketRange: bucketRanges[i],
		}
	}

	return allocations
}

// ValidateBucketingID validates that a bucketing ID is properly formatted
func (h *Hasher) ValidateBucketingID(bucketingID string) bool {
	// Should be a valid hex string
	_, err := hex.DecodeString(bucketingID)
	return err == nil && len(bucketingID) >= 8
}
