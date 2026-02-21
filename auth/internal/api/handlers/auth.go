package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/braccet/auth/internal/api/middleware"
	"github.com/braccet/auth/internal/domain"
	"github.com/braccet/auth/internal/repository"
	"github.com/braccet/auth/internal/service"
)

type AuthHandler struct {
	userRepo    repository.UserRepository
	pendingRepo repository.PendingRegistrationRepository
	emailSender service.EmailSender
	baseURL     string
}

func NewAuthHandler(
	userRepo repository.UserRepository,
	pendingRepo repository.PendingRegistrationRepository,
	emailSender service.EmailSender,
	baseURL string,
) *AuthHandler {
	return &AuthHandler{
		userRepo:    userRepo,
		pendingRepo: pendingRepo,
		emailSender: emailSender,
		baseURL:     baseURL,
	}
}

// SignupRequest represents the signup payload
type SignupRequest struct {
	Email       string `json:"email"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

// LoginRequest represents the login payload
type LoginRequest struct {
	Identifier string `json:"identifier"` // email or username
	Password   string `json:"password"`
}

// AuthResponse is returned on successful auth
type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         UserResponse `json:"user"`
}

// RefreshRequest represents the refresh token request payload
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshResponse is returned on successful token refresh
type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// UserResponse is the user data returned to clients
type UserResponse struct {
	ID          uint64  `json:"id"`
	Email       string  `json:"email"`
	Username    *string `json:"username,omitempty"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

// ErrorResponse for API errors
type ErrorResponse struct {
	Error string `json:"error"`
}

// MessageResponse for simple message responses
type MessageResponse struct {
	Message string `json:"message"`
}

// ResendVerificationRequest represents the resend verification payload
type ResendVerificationRequest struct {
	Email string `json:"email"`
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func validateEmail(email string) bool {
	return emailRegex.MatchString(email)
}

func validateUsername(username string) error {
	if len(username) < 3 {
		return errors.New("username must be at least 3 characters")
	}
	if len(username) > 50 {
		return errors.New("username must be at most 50 characters")
	}
	if !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(username) {
		return errors.New("username can only contain letters, numbers, and underscores")
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	return nil
}

// Signup handles user registration by creating a pending registration and sending verification email
func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	var req SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate input
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Username = strings.TrimSpace(req.Username)

	if !validateEmail(req.Email) {
		writeError(w, http.StatusBadRequest, "invalid email format")
		return
	}

	if err := validateUsername(req.Username); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := validatePassword(req.Password); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Check if username is taken (in users table)
	_, err := h.userRepo.GetByUsername(r.Context(), req.Username)
	if err == nil {
		writeError(w, http.StatusConflict, "username already taken")
		return
	}
	if !errors.Is(err, repository.ErrUserNotFound) {
		writeError(w, http.StatusInternalServerError, "failed to check username")
		return
	}

	// Check if email already verified (in users table)
	existingUser, err := h.userRepo.GetByEmail(r.Context(), req.Email)
	if err != nil && !errors.Is(err, repository.ErrUserNotFound) {
		writeError(w, http.StatusInternalServerError, "failed to check email")
		return
	}

	if existingUser != nil {
		// Email exists in users table
		// For security, return generic message to prevent enumeration
		// TODO: Handle OAuth account linking with verification
		writeJSON(w, http.StatusOK, MessageResponse{
			Message: "If this email is not already registered, you will receive a verification email shortly.",
		})
		return
	}

	// Hash password
	passwordHash, err := service.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to process request")
		return
	}

	// Generate verification token
	token, err := service.GenerateVerificationToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to process request")
		return
	}

	displayName := req.DisplayName
	if displayName == "" {
		displayName = req.Username
	}

	// Create pending registration (upserts if email exists in pending)
	pending := &domain.PendingRegistration{
		Email:             req.Email,
		Username:          req.Username,
		DisplayName:       displayName,
		PasswordHash:      passwordHash,
		VerificationToken: token,
		ExpiresAt:         time.Now().Add(service.GetTokenExpiry()),
	}

	if err := h.pendingRepo.Create(r.Context(), pending); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to process request")
		return
	}

	// Send verification email (async to not block response)
	go func() {
		verificationURL := service.BuildVerificationURL(h.baseURL, token)
		if err := h.emailSender.SendVerificationEmail(context.Background(), req.Email, verificationURL); err != nil {
			log.Printf("Failed to send verification email to %s: %v", req.Email, err)
		}
	}()

	// Always return same message to prevent email enumeration
	writeJSON(w, http.StatusOK, MessageResponse{
		Message: "If this email is not already registered, you will receive a verification email shortly.",
	})
}

// VerifyEmail handles email verification and creates the user account
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	// Token can come from query param (GET from email link) or body (POST)
	token := r.URL.Query().Get("token")
	if token == "" && r.Method == http.MethodPost {
		var body struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
			token = body.Token
		}
	}

	if token == "" {
		writeError(w, http.StatusBadRequest, "verification token required")
		return
	}

	// Look up pending registration
	pending, err := h.pendingRepo.GetByToken(r.Context(), token)
	if err != nil {
		if errors.Is(err, repository.ErrPendingRegistrationNotFound) {
			writeError(w, http.StatusBadRequest, "invalid or expired verification token")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to verify email")
		return
	}

	// Check expiration
	if pending.IsExpired() {
		// Clean up expired registration
		_ = h.pendingRepo.Delete(r.Context(), pending.ID)
		writeError(w, http.StatusBadRequest, "verification token has expired")
		return
	}

	// Re-check username availability (could have been taken while pending)
	_, err = h.userRepo.GetByUsername(r.Context(), pending.Username)
	if err == nil {
		writeError(w, http.StatusConflict, "username is no longer available")
		return
	}
	if !errors.Is(err, repository.ErrUserNotFound) {
		writeError(w, http.StatusInternalServerError, "failed to verify email")
		return
	}

	// Create the user
	user := &domain.User{
		Email:        pending.Email,
		DisplayName:  pending.DisplayName,
		Username:     &pending.Username,
		PasswordHash: &pending.PasswordHash,
	}

	if err := h.userRepo.Create(r.Context(), user); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create account")
		return
	}

	// Delete the pending registration
	if err := h.pendingRepo.Delete(r.Context(), pending.ID); err != nil {
		// Log but don't fail - user was created successfully
		log.Printf("Failed to delete pending registration %d: %v", pending.ID, err)
	}

	// Issue token pair
	tokens, err := service.IssueTokenPair(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "account created but failed to issue token")
		return
	}

	writeJSON(w, http.StatusCreated, AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User: UserResponse{
			ID:          user.ID,
			Email:       user.Email,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			AvatarURL:   user.AvatarURL,
		},
	})
}

// ResendVerification generates a new verification token and resends the email
func (h *AuthHandler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	var req ResendVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if !validateEmail(req.Email) {
		writeError(w, http.StatusBadRequest, "invalid email format")
		return
	}

	// Look up pending registration
	pending, err := h.pendingRepo.GetByEmail(r.Context(), req.Email)
	if err != nil {
		// Always return success to prevent enumeration
		writeJSON(w, http.StatusOK, MessageResponse{
			Message: "If a pending registration exists, a new verification email will be sent.",
		})
		return
	}

	// Generate new token
	token, err := service.GenerateVerificationToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to process request")
		return
	}

	// Update pending registration with new token and expiry
	pending.VerificationToken = token
	pending.ExpiresAt = time.Now().Add(service.GetTokenExpiry())

	if err := h.pendingRepo.UpdateToken(r.Context(), pending); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to process request")
		return
	}

	// Send new verification email
	go func() {
		verificationURL := service.BuildVerificationURL(h.baseURL, token)
		if err := h.emailSender.SendVerificationEmail(context.Background(), req.Email, verificationURL); err != nil {
			log.Printf("Failed to send verification email to %s: %v", req.Email, err)
		}
	}()

	writeJSON(w, http.StatusOK, MessageResponse{
		Message: "If a pending registration exists, a new verification email will be sent.",
	})
}

// Login handles user authentication
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Identifier = strings.TrimSpace(req.Identifier)

	if req.Identifier == "" {
		writeError(w, http.StatusBadRequest, "email or username required")
		return
	}

	if req.Password == "" {
		writeError(w, http.StatusBadRequest, "password required")
		return
	}

	// Try to find user by email first, then username
	var user *domain.User
	var err error

	if strings.Contains(req.Identifier, "@") {
		user, err = h.userRepo.GetByEmail(r.Context(), strings.ToLower(req.Identifier))
	} else {
		user, err = h.userRepo.GetByUsername(r.Context(), req.Identifier)
	}

	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to find user")
		return
	}

	// Check if user has password auth
	if user.PasswordHash == nil {
		writeError(w, http.StatusUnauthorized, "this account uses OAuth login only")
		return
	}

	// Verify password
	if !service.CheckPassword(*user.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// Issue token pair
	tokens, err := service.IssueTokenPair(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	writeJSON(w, http.StatusOK, AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User: UserResponse{
			ID:          user.ID,
			Email:       user.Email,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			AvatarURL:   user.AvatarURL,
		},
	})
}

// Me returns the current user's information
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	// Get user info from context (set by auth middleware)
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user not found in context")
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	writeJSON(w, http.StatusOK, UserResponse{
		ID:          user.ID,
		Email:       user.Email,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		AvatarURL:   user.AvatarURL,
	})
}

// Refresh exchanges a refresh token for a new access token
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh token required")
		return
	}

	// Validate the refresh token
	claims, err := service.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	// Get user to ensure they still exist
	user, err := h.userRepo.GetByID(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			writeError(w, http.StatusUnauthorized, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to verify user")
		return
	}

	// Issue new token pair
	tokens, err := service.IssueTokenPair(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue tokens")
		return
	}

	writeJSON(w, http.StatusOK, RefreshResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	})
}
