package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"loyalty/pkg/models"
	"loyalty/pkg/util"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// UserWithPendingPoints represents a user with their wallet and pending points count
type UserWithPendingPoints struct {
	User          interface{} `json:"user"`
	Wallet        interface{} `json:"wallet"`
	PendingPoints int         `json:"pendingPoints"`
}

// UsersWithPendingPointsResponse represents the response structure
type UsersWithPendingPointsResponse struct {
	Users      []UserWithPendingPoints `json:"users"`
	Pagination models.Pagination       `json:"pagination"`
}

// AuthUser represents user data from auth service
type AuthUser struct {
	ID                string      `json:"_id"`
	FirstName         string      `json:"firstName"`
	LastName          string      `json:"lastName"`
	Email             string      `json:"email"`
	Deactivated       bool        `json:"deactivated"`
	Disabled          bool        `json:"disabled"`
	EmailConfirmation bool        `json:"emailConfirmation"`
	Meta              []MetaField `json:"meta"`
	Gender            string      `json:"gender,omitempty"`
	Picture           string      `json:"picture,omitempty"`
	CreatedAt         string      `json:"createdAt"`
	UpdatedAt         string      `json:"updatedAt"`
	Roles             []string    `json:"roles"`
	Capabilities      []string    `json:"capabilities"`
}

type MetaField struct {
	ID    string `json:"_id"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

// AuthUsersResponse represents the response from auth service
type AuthUsersResponse struct {
	Users      []AuthUser        `json:"users"`
	Pagination models.Pagination `json:"pagination"`
}

// UserWallet represents wallet data
type UserWallet struct {
	ID             string `json:"_id"`
	Points         int    `json:"points"`
	UserID         string `json:"user_id"`
	UserName       string `json:"user_name"`
	Experience     int    `json:"experience"`
	UpdatedProfile bool   `json:"updatedProfile"`
	VisitCount     int    `json:"visitCount"`
	Code           string `json:"code"`
	CodeUsed       int    `json:"code_used"`
	CodeSent       int    `json:"code_sent"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
	CreatedBy      string `json:"createdBy"`
	UpdatedBy      string `json:"updatedBy"`
}

type UsersService interface {
	GetUsersWithPendingPoints(xclient string, args *util.ListArgs, pendingOnly bool, sortBy, sortOrder, minPendingPoints, maxPendingPoints, firstName, lastName, gender, email string) (*UsersWithPendingPointsResponse, error)
	GetAllUsersWithPendingPoints(xclient string, query string, pendingOnly bool, sortBy, sortOrder, minPendingPoints, maxPendingPoints, firstName, lastName, gender, email string) ([]UserWithPendingPoints, error)
}

type usersService struct {
	ctx context.Context
	db  string
}

func NewUsersService(ctx context.Context, db string) UsersService {
	return &usersService{
		ctx: ctx,
		db:  db,
	}
}

func (s *usersService) GetUsersWithPendingPoints(xclient string, args *util.ListArgs, pendingOnly bool, sortBy, sortOrder, minPendingPoints, maxPendingPoints, firstName, lastName, gender, email string) (*UsersWithPendingPointsResponse, error) {
	// Get access token from context
	accessToken := s.getAccessTokenFromContext()
	if accessToken == "" {
		return nil, fmt.Errorf("access token not found")
	}

	// Convert string parameters to int
	page, _ := strconv.Atoi(args.Page)
	perPage, _ := strconv.Atoi(args.PerPage)
	if perPage == 0 || args.PerPage == "" {
		perPage = 20 // Default per page
	}
	if page == 0 || args.Page == "" {
		page = 1 // Default to first page
	}

	// For filtered results, we need to fetch more data and then apply pagination
	// Fetch a larger set of users to account for filtering
	fetchLimit := perPage * 5 // Fetch 5x more to account for filtering
	if fetchLimit < 100 {
		fetchLimit = 100 // Minimum fetch size
	}

	// Fetch users from auth service with larger limit
	usersData, err := s.fetchUsersFromAuthService(xclient, accessToken, 1, fetchLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %v", err)
	}

	// Fetch wallets
	wallets, err := s.fetchWallets()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch wallets: %v", err)
	}

	// Aggregate users with pending points
	allUsers, err := s.aggregateUsersWithPendingPoints(usersData.Users, wallets)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate pending points: %v", err)
	}

	// Apply filtering
	filteredUsers := allUsers
	if pendingOnly {
		filteredUsers = s.filterUsersByPendingPoints(filteredUsers)
	}

	// Apply pending points range filtering
	filteredUsers = s.filterUsersByPendingPointsRange(filteredUsers, minPendingPoints, maxPendingPoints)

	// Apply user field filtering
	filteredUsers = s.filterUsersByFields(filteredUsers, firstName, lastName, gender, email)

	// Apply sorting
	filteredUsers = s.sortUsers(filteredUsers, sortBy, sortOrder)

	// Calculate pagination based on filtered results
	totalCount := int64(len(filteredUsers))
	perPageInt64 := int64(perPage)
	totalPages := float64(totalCount) / float64(perPageInt64)
	if totalCount%perPageInt64 > 0 {
		totalPages = float64(int(totalPages) + 1)
	}

	// Apply pagination to filtered results
	startIndex := (page - 1) * perPage
	endIndex := startIndex + perPage
	if startIndex >= len(filteredUsers) {
		// Return empty result if page is beyond available data
		return &UsersWithPendingPointsResponse{
			Users: []UserWithPendingPoints{},
			Pagination: models.Pagination{
				TotalPages: totalPages,
				PerPage:    perPageInt64,
				TotalCount: totalCount,
			},
		}, nil
	}
	if endIndex > len(filteredUsers) {
		endIndex = len(filteredUsers)
	}

	paginatedUsers := filteredUsers[startIndex:endIndex]

	updatedPagination := models.Pagination{
		TotalPages: totalPages,
		PerPage:    perPageInt64,
		TotalCount: totalCount,
	}

	return &UsersWithPendingPointsResponse{
		Users:      paginatedUsers,
		Pagination: updatedPagination,
	}, nil
}

func (s *usersService) GetAllUsersWithPendingPoints(xclient, query string, pendingOnly bool, sortBy, sortOrder, minPendingPoints, maxPendingPoints, firstName, lastName, gender, email string) ([]UserWithPendingPoints, error) {
	// Get access token from context
	accessToken := s.getAccessTokenFromContext()
	if accessToken == "" {
		return nil, fmt.Errorf("access token not found")
	}

	// Debug: Log token extraction (first 10 chars for security)
	fmt.Printf("Debug: Access token extracted (first 10 chars): %s...\n", accessToken[:min(10, len(accessToken))])

	// Fetch all users from auth service
	users, err := s.fetchAllUsersFromAuthService(xclient, accessToken, query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %v", err)
	}

	// Fetch wallets
	wallets, err := s.fetchWallets()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch wallets: %v", err)
	}

	// Aggregate users with pending points
	result, err := s.aggregateUsersWithPendingPoints(users, wallets)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate pending points: %v", err)
	}

	// Apply filtering
	if pendingOnly {
		result = s.filterUsersByPendingPoints(result)
	}

	// Apply pending points range filtering
	result = s.filterUsersByPendingPointsRange(result, minPendingPoints, maxPendingPoints)

	// Apply user field filtering
	result = s.filterUsersByFields(result, firstName, lastName, gender, email)

	// Apply sorting
	result = s.sortUsers(result, sortBy, sortOrder)

	return result, nil
}

func (s *usersService) getAccessTokenFromContext() string {
	// Extract access token from context
	if token := s.ctx.Value("x-access-token"); token != nil {
		if tokenStr, ok := token.(string); ok {
			return tokenStr
		}
	}

	// Try to get from Authorization header context
	if auth := s.ctx.Value("Authorization"); auth != nil {
		if authStr, ok := auth.(string); ok {
			if strings.HasPrefix(authStr, "Bearer ") {
				return strings.TrimPrefix(authStr, "Bearer ")
			}
			return authStr
		}
	}

	return ""
}

func (s *usersService) fetchUsersFromAuthService(xclient, accessToken string, page, limit int) (*AuthUsersResponse, error) {
	// Build URL with query parameters
	url := "https://services.larsa.io/auth/users/all"
	if page > 0 || limit > 0 {
		url += "?"
		params := []string{}
		if page > 0 {
			params = append(params, "page="+strconv.Itoa(page))
		}
		if limit > 0 {
			params = append(params, "limit="+strconv.Itoa(limit))
		}
		url += strings.Join(params, "&")
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-access-token", accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(("x-client"), xclient)
	// Debug: Log request details
	fmt.Printf("Debug: Making request to: %s\n", url)
	fmt.Printf("Debug: Access token header (first 10 chars): %s...\n", accessToken[:min(10, len(accessToken))])

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Debug: Request failed with error: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Debug: Log response status
	fmt.Printf("Debug: Auth service response status: %d\n", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Debug: Auth service error response: %s\n", string(body))
		return nil, fmt.Errorf("auth service returned status %d: %s", resp.StatusCode, string(body))
	}

	var result AuthUsersResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *usersService) fetchAllUsersFromAuthService(xclient, accessToken, query string) ([]AuthUser, error) {
	// Build URL with query parameters
	url := "https://services.larsa.io/auth/users/all"
	if query != "" {
		url += "?query=" + query
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-access-token", accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(("x-client"), xclient)
	// Debug: Log request details
	fmt.Printf("Debug: Making request to: %s\n", url)
	fmt.Printf("Debug: Access token header (first 10 chars): %s...\n", accessToken[:min(10, len(accessToken))])

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Debug: Request failed with error: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Debug: Log response status
	fmt.Printf("Debug: Auth service response status: %d\n", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Debug: Auth service error response: %s\n", string(body))
		return nil, fmt.Errorf("auth service returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Users []AuthUser `json:"users"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Users, nil
}

func (s *usersService) fetchWallets() ([]UserWallet, error) {
	// Use the existing userwallets service to get wallets
	userWalletService := NewUserWalletSvcs(s.ctx, s.db)
	wallets, err := userWalletService.GetAll("")
	if err != nil {
		return nil, err
	}

	// Convert to UserWallet slice
	result := make([]UserWallet, len(wallets))
	for i, wallet := range wallets {
		result[i] = UserWallet{
			ID:             wallet.Id.Hex(),
			Points:         int(wallet.Points),
			UserID:         wallet.UserId.Hex(),
			UserName:       wallet.UserName,
			Experience:     int(wallet.Experience),
			UpdatedProfile: wallet.UpdatedProfile,
			VisitCount:     int(wallet.VisitCount),
			Code:           wallet.Code,
			CodeUsed:       int(wallet.CodeUsed),
			CodeSent:       int(wallet.CodeSent),
			CreatedAt:      wallet.CreatedAt.Format("2006-01-02T15:04:05.000Z"),
			UpdatedAt:      wallet.UpdatedAt.Format("2006-01-02T15:04:05.000Z"),
		}
		if wallet.CreatedBy != nil {
			result[i].CreatedBy = wallet.CreatedBy.Hex()
		}
		if wallet.UpdatedBy != nil {
			result[i].UpdatedBy = wallet.UpdatedBy.Hex()
		}
	}

	return result, nil
}

func (s *usersService) aggregateUsersWithPendingPoints(users []AuthUser, wallets []UserWallet) ([]UserWithPendingPoints, error) {
	result := make([]UserWithPendingPoints, 0, len(users))

	for _, user := range users {
		// Find wallet for this user
		var userWallet *UserWallet
		for _, wallet := range wallets {
			if wallet.UserID == user.ID {
				userWallet = &wallet
				break
			}
		}

		// Get pending points count for this user
		pendingPoints, err := s.getPendingPointsForUser(user.ID)
		if err != nil {
			// Log error but continue with 0 pending points
			fmt.Printf("Error getting pending points for user %s: %v\n", user.ID, err)
			pendingPoints = 0
		}

		result = append(result, UserWithPendingPoints{
			User:          user,
			Wallet:        userWallet,
			PendingPoints: pendingPoints,
		})
	}

	return result, nil
}

func (s *usersService) getPendingPointsForUser(userID string) (int, error) {
	// Get challenge answers service
	challengeAnswerService := NewChallengeAnswerSvcs(s.ctx, s.db)

	// Convert userID string to ObjectID
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return 0, fmt.Errorf("invalid user ID: %v", err)
	}

	// Build query to get non-accepted challenge answers for this user
	// Use ObjectID for user_id field
	query := fmt.Sprintf(`{"user_id":{"$oid":"%s"},"is_accepted":false,"trash":{"$ne":true}}`, userObjectID.Hex())

	// Debug: Log the query being used
	fmt.Printf("Debug: Challenge answer query: %s\n", query)

	challengeAnswers, err := challengeAnswerService.GetAll(query)
	if err != nil {
		fmt.Printf("Debug: Error fetching challenge answers: %v\n", err)
		return 0, err
	}

	// Debug: Log the number of challenge answers found
	fmt.Printf("Debug: Found %d challenge answers for user %s\n", len(challengeAnswers), userID)

	// Count non-trashed challenge answers
	pendingCount := 0
	for _, answer := range challengeAnswers {
		if !answer.Trash {
			pendingCount++
		}
	}

	fmt.Printf("Debug: Pending points count for user %s: %d\n", userID, pendingCount)
	return pendingCount, nil
}

func (s *usersService) filterUsersByPendingPoints(users []UserWithPendingPoints) []UserWithPendingPoints {
	var filtered []UserWithPendingPoints
	for _, user := range users {
		if user.PendingPoints > 0 {
			filtered = append(filtered, user)
		}
	}
	return filtered
}

func (s *usersService) filterUsersByPendingPointsRange(users []UserWithPendingPoints, minPendingPoints, maxPendingPoints string) []UserWithPendingPoints {
	// If no range filters are provided, return all users
	if minPendingPoints == "" && maxPendingPoints == "" {
		return users
	}

	var filtered []UserWithPendingPoints
	for _, user := range users {
		includeUser := true

		// Check minimum pending points
		if minPendingPoints != "" {
			if minPoints, err := strconv.Atoi(minPendingPoints); err == nil {
				if user.PendingPoints < minPoints {
					includeUser = false
				}
			}
		}

		// Check maximum pending points
		if maxPendingPoints != "" && includeUser {
			if maxPoints, err := strconv.Atoi(maxPendingPoints); err == nil {
				if user.PendingPoints > maxPoints {
					includeUser = false
				}
			}
		}

		if includeUser {
			filtered = append(filtered, user)
		}
	}

	return filtered
}

func (s *usersService) filterUsersByFields(users []UserWithPendingPoints, firstName, lastName, gender, email string) []UserWithPendingPoints {
	// If no field filters are provided, return all users
	if firstName == "" && lastName == "" && gender == "" && email == "" {
		return users
	}

	var filtered []UserWithPendingPoints
	for _, userWithPending := range users {
		user, ok := userWithPending.User.(AuthUser)
		if !ok {
			continue
		}

		includeUser := true

		// Filter by firstName (case-insensitive partial match)
		if firstName != "" {
			if !strings.Contains(strings.ToLower(user.FirstName), strings.ToLower(firstName)) {
				includeUser = false
			}
		}

		// Filter by lastName (case-insensitive partial match)
		if lastName != "" && includeUser {
			if !strings.Contains(strings.ToLower(user.LastName), strings.ToLower(lastName)) {
				includeUser = false
			}
		}

		// Filter by gender (case-insensitive exact match)
		if gender != "" && includeUser {
			if !strings.EqualFold(user.Gender, gender) {
				includeUser = false
			}
		}

		// Filter by email (case-insensitive partial match)
		if email != "" && includeUser {
			if !strings.Contains(strings.ToLower(user.Email), strings.ToLower(email)) {
				includeUser = false
			}
		}

		if includeUser {
			filtered = append(filtered, userWithPending)
		}
	}

	return filtered
}

func (s *usersService) sortUsers(users []UserWithPendingPoints, sortBy, sortOrder string) []UserWithPendingPoints {
	if sortBy == "" {
		return users
	}

	sort.Slice(users, func(i, j int) bool {
		switch sortBy {
		case "pendingPoints":
			if sortOrder == "asc" {
				return users[i].PendingPoints < users[j].PendingPoints
			}
			return users[i].PendingPoints > users[j].PendingPoints
		case "name":
			userI, okI := users[i].User.(AuthUser)
			userJ, okJ := users[j].User.(AuthUser)
			if !okI || !okJ {
				return false
			}
			nameI := userI.FirstName + " " + userI.LastName
			nameJ := userJ.FirstName + " " + userJ.LastName
			if sortOrder == "asc" {
				return nameI < nameJ
			}
			return nameI > nameJ
		case "firstName":
			userI, okI := users[i].User.(AuthUser)
			userJ, okJ := users[j].User.(AuthUser)
			if !okI || !okJ {
				return false
			}
			if sortOrder == "asc" {
				return userI.FirstName < userJ.FirstName
			}
			return userI.FirstName > userJ.FirstName
		case "lastName":
			userI, okI := users[i].User.(AuthUser)
			userJ, okJ := users[j].User.(AuthUser)
			if !okI || !okJ {
				return false
			}
			if sortOrder == "asc" {
				return userI.LastName < userJ.LastName
			}
			return userI.LastName > userJ.LastName
		case "email":
			userI, okI := users[i].User.(AuthUser)
			userJ, okJ := users[j].User.(AuthUser)
			if !okI || !okJ {
				return false
			}
			if sortOrder == "asc" {
				return userI.Email < userJ.Email
			}
			return userI.Email > userJ.Email
		default:
			return false
		}
	})

	return users
}
