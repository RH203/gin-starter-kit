package handler

import (
	"errors"
	"net/http"
	"strconv"

	"gin-starter-pack/internal/domain"
	"gin-starter-pack/pkg/response"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	usecase      domain.UserUsecase
	cookieMaxAge int
}

func NewUserHandler(usecase domain.UserUsecase, cookieMaxAge ...int) *UserHandler {
	maxAge := 86400
	if len(cookieMaxAge) > 0 && cookieMaxAge[0] > 0 {
		maxAge = cookieMaxAge[0]
	}
	return &UserHandler{
		usecase:      usecase,
		cookieMaxAge: maxAge,
	}
}

// Create godoc
// @Summary      Register new user
// @Description  Create a new user account with hashed password
// @Tags         Auth, Users
// @Accept       json
// @Produce      json
// @Param        request body domain.CreateUserRequest true "User Registration Payload"
// @Success      201  {object}  response.APIResponse
// @Failure      400  {object}  response.APIResponse
// @Failure      409  {object}  response.APIResponse
// @Router       /auth/register [post]
// @Router       /users [post]
func (h *UserHandler) Create(c *gin.Context) {
	var req domain.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	user, err := h.usecase.Register(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			response.Error(c, http.StatusConflict, "Email already registered", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to create user", err.Error())
		return
	}

	response.Created(c, "User created successfully", user)
}

// Login godoc
// @Summary      User login
// @Description  Authenticate user with email and password to receive a signed JWT token and auth cookie
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body domain.LoginRequest true "Login Credentials"
// @Success      200  {object}  response.APIResponse
// @Failure      400  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Router       /auth/login [post]
func (h *UserHandler) Login(c *gin.Context) {
	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid login credentials format", err.Error())
		return
	}

	loginResp, err := h.usecase.Login(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			response.Error(c, http.StatusUnauthorized, "Invalid email or password", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to process login", err.Error())
		return
	}

	// Set HTTP-only Cookie for web client authentication
	isSecure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access_token", loginResp.Token, h.cookieMaxAge, "/", "", isSecure, true)
	c.SetCookie("token", loginResp.Token, h.cookieMaxAge, "/", "", isSecure, true)

	response.OK(c, "Login successful", loginResp)
}

// Logout godoc
// @Summary      User logout
// @Description  Clear the authentication cookies
// @Tags         Auth
// @Produce      json
// @Success      200  {object}  response.APIResponse
// @Router       /auth/logout [post]
func (h *UserHandler) Logout(c *gin.Context) {
	isSecure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access_token", "", -1, "/", "", isSecure, true)
	c.SetCookie("token", "", -1, "/", "", isSecure, true)

	response.OK(c, "Logout successful", nil)
}

// GetProfile godoc
// @Summary      Get current user profile
// @Description  Retrieve the profile of the currently authenticated user
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Router       /auth/me [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "Unauthorized access", nil)
		return
	}

	profile, err := h.usecase.GetProfile(c.Request.Context(), userID.(string))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.Error(c, http.StatusNotFound, "User not found", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to retrieve profile", err.Error())
		return
	}

	response.OK(c, "Profile retrieved successfully", profile)
}

// GetByID godoc
// @Summary      Get user by ID
// @Description  Retrieve user detail by UUID (cached in Redis if enabled)
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "User ID"
// @Success      200  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Router       /users/{id} [get]
func (h *UserHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "User ID is required", nil)
		return
	}

	user, err := h.usecase.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.Error(c, http.StatusNotFound, "User not found", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to retrieve user", err.Error())
		return
	}

	response.OK(c, "User retrieved successfully", user)
}

// GetAll godoc
// @Summary      List users with pagination
// @Description  Get paginated list of users
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        page      query     int  false  "Page number (default 1)"
// @Param        page_size query     int  false  "Page size (default 10)"
// @Success      200  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Router       /users [get]
func (h *UserHandler) GetAll(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	users, total, err := h.usecase.GetAll(c.Request.Context(), page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to list users", err.Error())
		return
	}

	response.Paginated(c, "Users retrieved successfully", users, total, page, pageSize)
}

// Update godoc
// @Summary      Update user
// @Description  Update user profile details
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path      string                   true  "User ID"
// @Param        request body      domain.UpdateUserRequest true  "Update payload"
// @Success      200  {object}  response.APIResponse
// @Failure      400  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Failure      409  {object}  response.APIResponse
// @Router       /users/{id} [put]
func (h *UserHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req domain.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	user, err := h.usecase.Update(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.Error(c, http.StatusNotFound, "User not found", nil)
			return
		}
		if errors.Is(err, domain.ErrAlreadyExists) {
			response.Error(c, http.StatusConflict, "Email is already taken", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to update user", err.Error())
		return
	}

	response.OK(c, "User updated successfully", user)
}

// Delete godoc
// @Summary      Delete user
// @Description  Delete user by UUID
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "User ID"
// @Success      200  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Router       /users/{id} [delete]
func (h *UserHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.usecase.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.Error(c, http.StatusNotFound, "User not found", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to delete user", err.Error())
		return
	}

	response.OK(c, "User deleted successfully", nil)
}
