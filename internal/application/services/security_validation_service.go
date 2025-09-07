/*
 * SQL Graph Visualizer - Security Validation Service
 *
 * Copyright (c) 2024
 * Licensed under Dual License: AGPL-3.0 OR Commercial License
 * See LICENSE file for details
 * Patent Pending - Application filed for innovative database transformation techniques
 */

package services

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"sql-graph-visualizer/internal/domain/models"
)

// SecurityValidationService provides comprehensive security validation
// for database connections and operations
type SecurityValidationService struct {
	config *models.SecurityConfig
}

// NewSecurityValidationService creates a new security validation service
func NewSecurityValidationService(config *models.SecurityConfig) *SecurityValidationService {
	return &SecurityValidationService{
		config: config,
	}
}

// ValidateConnectionSecurity performs comprehensive security validation
// on database connection parameters
func (s *SecurityValidationService) ValidateConnectionSecurity(
	ctx context.Context,
	dbConfig *models.MySQLConfig,
) (*models.SecurityValidationResult, error) {

	result := &models.SecurityValidationResult{
		IsValid:         true,
		SecurityLevel:   "HIGH",
		Validations:     make(map[string]*models.ValidationCheck),
		Recommendations: []string{},
	}

	// Step 1: Validate connection parameters
	s.validateConnectionParameters(dbConfig, result)

	// Step 2: Validate network security
	err := s.validateNetworkSecurity(ctx, dbConfig, result)
	if err != nil {
		return nil, fmt.Errorf("network security validation failed: %w", err)
	}

	// Step 3: Validate SSL/TLS configuration
	s.validateSSLConfiguration(dbConfig, result)

	// Step 4: Validate authentication method
	s.validateAuthenticationMethod(dbConfig, result)

	// Step 5: Check against security policies
	s.applySecurityPolicies(dbConfig, result)

	// Calculate final security level
	s.calculateSecurityLevel(result)

	return result, nil
}

// validateConnectionParameters validates basic connection security parameters
func (s *SecurityValidationService) validateConnectionParameters(
	dbConfig *models.MySQLConfig,
	result *models.SecurityValidationResult,
) {

	// Validate host security
	hostCheck := &models.ValidationCheck{
		CheckName:   "host_security",
		Passed:      true,
		Severity:    "HIGH",
		Description: "Database host security validation",
	}

	if s.isProductionHost(dbConfig.Host) && !s.config.AllowProductionConnections {
		hostCheck.Passed = false
		hostCheck.Message = "Production database connections are not allowed"
		result.IsValid = false
	} else if s.isLocalhostConnection(dbConfig.Host) {
		hostCheck.Message = "Localhost connection detected - ensure proper authentication"
		hostCheck.Severity = "MEDIUM"
	} else {
		hostCheck.Message = "Host validation passed"
	}

	result.Validations["host_security"] = hostCheck

	// Validate port security
	portCheck := &models.ValidationCheck{
		CheckName:   "port_security",
		Passed:      true,
		Severity:    "MEDIUM",
		Description: "Database port security validation",
	}

	if dbConfig.Port != 3306 && dbConfig.Port != 3307 {
		portCheck.Message = "Non-standard MySQL port detected"
		portCheck.Severity = "LOW"
	} else {
		portCheck.Message = "Standard MySQL port in use"
	}

	result.Validations["port_security"] = portCheck

	// Validate credentials
	credCheck := &models.ValidationCheck{
		CheckName:   "credentials_security",
		Passed:      true,
		Severity:    "CRITICAL",
		Description: "Database credentials security validation",
	}

	if s.isWeakPassword(dbConfig.Password) {
		credCheck.Passed = false
		credCheck.Message = "Weak password detected - use strong passwords for production"
		credCheck.Severity = "CRITICAL"
		result.IsValid = false
		result.Recommendations = append(result.Recommendations,
			"Use strong passwords with mixed case, numbers, and special characters")
	} else if s.isDefaultCredentials(dbConfig.Username, dbConfig.Password) {
		credCheck.Passed = false
		credCheck.Message = "Default credentials detected"
		credCheck.Severity = "CRITICAL"
		result.IsValid = false
		result.Recommendations = append(result.Recommendations,
			"Change default database credentials immediately")
	} else {
		credCheck.Message = "Credentials validation passed"
	}

	result.Validations["credentials_security"] = credCheck
}

// validateNetworkSecurity performs network-level security validation
func (s *SecurityValidationService) validateNetworkSecurity(
	ctx context.Context,
	dbConfig *models.MySQLConfig,
	result *models.SecurityValidationResult,
) error {

	networkCheck := &models.ValidationCheck{
		CheckName:   "network_security",
		Passed:      true,
		Severity:    "HIGH",
		Description: "Network connectivity and security validation",
	}

	// Test network connectivity with timeout
	timeout := 5 * time.Second
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", dbConfig.Host, dbConfig.Port), timeout)
	if err != nil {
		networkCheck.Passed = false
		networkCheck.Message = fmt.Sprintf("Cannot establish network connection: %v", err)
		result.IsValid = false
		result.Validations["network_security"] = networkCheck
		return nil
	}
	defer conn.Close()

	// Check if connection is over public network
	if s.isPublicIP(dbConfig.Host) && !dbConfig.SSLConfig.Enabled {
		networkCheck.Passed = false
		networkCheck.Message = "Unencrypted connection over public network"
		networkCheck.Severity = "CRITICAL"
		result.IsValid = false
		result.Recommendations = append(result.Recommendations,
			"Enable SSL/TLS for connections over public networks")
	} else {
		networkCheck.Message = "Network connectivity validation passed"
	}

	result.Validations["network_security"] = networkCheck
	return nil
}

// validateSSLConfiguration validates SSL/TLS security settings
func (s *SecurityValidationService) validateSSLConfiguration(
	dbConfig *models.MySQLConfig,
	result *models.SecurityValidationResult,
) {

	sslCheck := &models.ValidationCheck{
		CheckName:   "ssl_security",
		Passed:      true,
		Severity:    "HIGH",
		Description: "SSL/TLS configuration validation",
	}

	if !dbConfig.SSLConfig.Enabled {
		if s.isProductionHost(dbConfig.Host) || s.isPublicIP(dbConfig.Host) {
			sslCheck.Passed = false
			sslCheck.Message = "SSL/TLS is required for production or public network connections"
			sslCheck.Severity = "CRITICAL"
			result.IsValid = false
		} else {
			sslCheck.Message = "SSL/TLS disabled - acceptable for local development"
			sslCheck.Severity = "MEDIUM"
		}
		result.Recommendations = append(result.Recommendations,
			"Enable SSL/TLS encryption for enhanced security")
	} else {
		// Validate SSL certificate settings
		if dbConfig.SSLConfig.InsecureSkipVerify {
			sslCheck.Message = "SSL certificate verification is disabled - security risk"
			sslCheck.Severity = "HIGH"
			result.Recommendations = append(result.Recommendations,
				"Enable SSL certificate verification in production")
		} else {
			sslCheck.Message = "SSL/TLS configuration is secure"
		}

		// Check certificate files if provided
		if dbConfig.SSLConfig.CertFile != "" || dbConfig.SSLConfig.KeyFile != "" {
			if err := s.validateSSLCertificates(dbConfig.SSLConfig); err != nil {
				sslCheck.Passed = false
				sslCheck.Message = fmt.Sprintf("SSL certificate validation failed: %v", err)
				result.IsValid = false
			}
		}
	}

	result.Validations["ssl_security"] = sslCheck
}

// validateAuthenticationMethod validates database authentication security
func (s *SecurityValidationService) validateAuthenticationMethod(
	dbConfig *models.MySQLConfig,
	result *models.SecurityValidationResult,
) {

	authCheck := &models.ValidationCheck{
		CheckName:   "authentication_security",
		Passed:      true,
		Severity:    "HIGH",
		Description: "Database authentication method validation",
	}

	// Check for root user usage
	if strings.ToLower(dbConfig.Username) == "root" && !s.config.AllowRootUser {
		authCheck.Passed = false
		authCheck.Message = "Root user access is not recommended for applications"
		authCheck.Severity = "HIGH"
		result.Recommendations = append(result.Recommendations,
			"Create dedicated database user with minimal required privileges")
	} else {
		authCheck.Message = "Authentication method validation passed"
	}

	result.Validations["authentication_security"] = authCheck
}

// applySecurityPolicies applies configured security policies
func (s *SecurityValidationService) applySecurityPolicies(
	dbConfig *models.MySQLConfig,
	result *models.SecurityValidationResult,
) {

	policyCheck := &models.ValidationCheck{
		CheckName:   "security_policies",
		Passed:      true,
		Severity:    "MEDIUM",
		Description: "Security policy compliance validation",
	}

	var violations []string

	// Check allowed hosts policy
	if len(s.config.AllowedHosts) > 0 {
		allowed := false
		for _, allowedHost := range s.config.AllowedHosts {
			if s.matchesHostPattern(dbConfig.Host, allowedHost) {
				allowed = true
				break
			}
		}
		if !allowed {
			violations = append(violations, "Host not in allowed hosts list")
		}
	}

	// Check forbidden patterns
	for _, pattern := range s.config.ForbiddenPatterns {
		if s.matchesPattern(dbConfig.Host, pattern) {
			violations = append(violations, fmt.Sprintf("Host matches forbidden pattern: %s", pattern))
		}
	}

	if len(violations) > 0 {
		policyCheck.Passed = false
		policyCheck.Message = strings.Join(violations, "; ")
		result.IsValid = false
	} else {
		policyCheck.Message = "Security policy compliance verified"
	}

	result.Validations["security_policies"] = policyCheck
}

// calculateSecurityLevel determines overall security level based on validations
func (s *SecurityValidationService) calculateSecurityLevel(result *models.SecurityValidationResult) {
	highSeverityIssues := 0
	criticalIssues := 0

	for _, validation := range result.Validations {
		if !validation.Passed {
			switch validation.Severity {
			case "CRITICAL":
				criticalIssues++
			case "HIGH":
				highSeverityIssues++
			}
		}
	}

	if criticalIssues > 0 {
		result.SecurityLevel = "CRITICAL_RISK"
	} else if highSeverityIssues > 0 {
		result.SecurityLevel = "HIGH_RISK"
	} else if highSeverityIssues == 0 && len(result.Recommendations) > 0 {
		result.SecurityLevel = "MEDIUM"
	} else {
		result.SecurityLevel = "HIGH"
	}
}

// Helper functions for security validation

func (s *SecurityValidationService) isProductionHost(host string) bool {
	prodPatterns := []string{
		`.*prod.*`,
		`.*production.*`,
		`.*live.*`,
		`.*master.*`,
	}

	hostLower := strings.ToLower(host)
	for _, pattern := range prodPatterns {
		matched, _ := regexp.MatchString(pattern, hostLower)
		if matched {
			return true
		}
	}
	return false
}

func (s *SecurityValidationService) isLocalhostConnection(host string) bool {
	localhost := []string{"localhost", "127.0.0.1", "::1", "0.0.0.0"}
	for _, local := range localhost {
		if host == local {
			return true
		}
	}
	return false
}

func (s *SecurityValidationService) isPublicIP(host string) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		// If it's not an IP, assume it's a hostname that could be public
		return !s.isLocalhostConnection(host)
	}

	return !ip.IsLoopback() && !ip.IsPrivate()
}

func (s *SecurityValidationService) isWeakPassword(password string) bool {
	if len(password) < 8 {
		return true
	}

	// Check for common weak patterns
	weakPatterns := []string{
		`^password`,
		`^123`,
		`^admin`,
		`^root`,
		`^test`,
	}

	passwordLower := strings.ToLower(password)
	for _, pattern := range weakPatterns {
		matched, _ := regexp.MatchString(pattern, passwordLower)
		if matched {
			return true
		}
	}

	return false
}

func (s *SecurityValidationService) isDefaultCredentials(username, password string) bool {
	// Check for common default username/password combinations
	defaultCombos := []struct {
		username string
		password string
	}{
		{"root", "root"},
		{"root", "password"},
		{"root", "admin"},
		{"admin", "admin"},
		{"admin", "password"},
		{"test", "test"},
	}

	usernameLower := strings.ToLower(username)
	passwordLower := strings.ToLower(password)

	for _, combo := range defaultCombos {
		if combo.username == usernameLower && combo.password == passwordLower {
			return true
		}
	}

	return false
}

func (s *SecurityValidationService) validateSSLCertificates(sslConfig models.SSLConfig) error {
	if sslConfig.CertFile != "" {
		cert, err := tls.LoadX509KeyPair(sslConfig.CertFile, sslConfig.KeyFile)
		if err != nil {
			return fmt.Errorf("failed to load SSL certificate: %w", err)
		}

		// Parse certificate to check validity
		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			return fmt.Errorf("failed to parse SSL certificate: %w", err)
		}

		// Check if certificate is expired
		if time.Now().After(x509Cert.NotAfter) {
			return fmt.Errorf("SSL certificate has expired")
		}

		// Check if certificate is not yet valid
		if time.Now().Before(x509Cert.NotBefore) {
			return fmt.Errorf("SSL certificate is not yet valid")
		}
	}

	return nil
}

func (s *SecurityValidationService) matchesHostPattern(host, pattern string) bool {
	matched, _ := regexp.MatchString(pattern, host)
	return matched
}

func (s *SecurityValidationService) matchesPattern(input, pattern string) bool {
	matched, _ := regexp.MatchString(pattern, input)
	return matched
}
