package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/MSSkowron/BookRESTAPI/internal/models"
	"github.com/MSSkowron/BookRESTAPI/internal/services"
	"github.com/gorilla/mux"
)

const (
	// ErrMsgBadRequestInvalidRequestBody is a message for bad request with invalid request body
	ErrMsgBadRequestInvalidRequestBody = "invalid request body"
	// ErrMsgBadRequestUserAlreadyExists is a message for bad request with user already exists
	ErrMsgBadRequestUserAlreadyExists = "user already exists"
	// ErrMsgBadRequestInvalidBookID is a message for bad request with invalid book id
	ErrMsgBadRequestInvalidBookID = "invalid book id"
	// ErrMsgUnauthorized is a message for unauthorized
	ErrMsgUnauthorized = "unauthorized"
	// ErrMsgUnauthorizedExpiredToken is a message for unauthorized with expired token
	ErrMsgUnauthorizedExpiredToken = "expired token"
	// ErrMsgUnauthorizedInvalidToken is a message for unauthorized with invalid token
	ErrMsgUnauthorizedInvalidToken = "invalid token"
	// ErrMsgUnauthorizedInvalidCredentials is a message for unauthorized with invalid credentials
	ErrMsgUnauthorizedInvalidCredentials = "invalid credentials"
	// ErrMsgNotFound is a message for not found
	ErrMsgNotFound = "not found"
	// ErrMsgInternalError is a message for internal error
	ErrMsgInternalError = "internal error"
)

type ServerHandlerFunc func(w http.ResponseWriter, r *http.Request) error

func makeHTTPHandlerFunc(f ServerHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			log.Printf("[Server] Error while handling request: %v", err)
		}
	}
}

// Server is a HTTP server for handling REST API requests
type Server struct {
	listenAddr  string
	userService services.UserService
	bookService services.BookService
}

// NewServer creates a new Server
func NewServer(listenAddr string, userService services.UserService, bookService services.BookService) *Server {
	return &Server{
		listenAddr:  listenAddr,
		userService: userService,
		bookService: bookService,
	}
}

// Run runs the Server
func (s *Server) Run() error {
	r := mux.NewRouter()
	r.HandleFunc("/register", makeHTTPHandlerFunc(s.handleRegister)).Methods("POST")
	r.HandleFunc("/login", makeHTTPHandlerFunc(s.handleLogin)).Methods("POST")
	r.HandleFunc("/books", s.validateJWT(makeHTTPHandlerFunc(s.handleGetBooks))).Methods("GET")
	r.HandleFunc("/books", s.validateJWT(makeHTTPHandlerFunc(s.handlePostBook))).Methods("POST")
	r.HandleFunc("/books/{id}", s.validateJWT(makeHTTPHandlerFunc(s.handleGetBookByID))).Methods("GET")
	r.HandleFunc("/books/{id}", s.validateJWT(makeHTTPHandlerFunc(s.handlePutBookByID))).Methods("PUT")
	r.HandleFunc("/books/{id}", s.validateJWT(makeHTTPHandlerFunc(s.handleDeleteBookByID))).Methods("DELETE")

	log.Println("[Server] Server is running on: " + s.listenAddr)

	return http.ListenAndServe(s.listenAddr, r)
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) error {
	log.Println("[Server] Called POST /register")

	createAccountRequest := &models.CreateAccountRequest{}
	if err := json.NewDecoder(r.Body).Decode(createAccountRequest); err != nil {
		s.respondWithError(w, http.StatusBadRequest, ErrMsgBadRequestInvalidRequestBody)
		return nil
	}

	user, err := s.userService.RegisterUser(createAccountRequest.Email, createAccountRequest.Password, createAccountRequest.FirstName, createAccountRequest.LastName, int(createAccountRequest.Age))
	if err != nil {
		if errors.Is(err, services.ErrUserAlreadyExists) {
			s.respondWithError(w, http.StatusBadRequest, ErrMsgBadRequestUserAlreadyExists)
			return nil
		}
		if errors.Is(err, services.ErrInvalidEmail) {
			s.respondWithError(w, http.StatusBadRequest, ErrMsgBadRequestInvalidRequestBody)
			return nil
		}
		if errors.Is(err, services.ErrInvalidPassword) {
			s.respondWithError(w, http.StatusBadRequest, ErrMsgBadRequestInvalidRequestBody)
			return nil
		}
		if errors.Is(err, services.ErrInvalidFirstName) {
			s.respondWithError(w, http.StatusBadRequest, ErrMsgBadRequestInvalidRequestBody)
			return nil
		}
		if errors.Is(err, services.ErrInvalidLastName) {
			s.respondWithError(w, http.StatusBadRequest, ErrMsgBadRequestInvalidRequestBody)
			return nil
		}
		if errors.Is(err, services.ErrInvalidAge) {
			s.respondWithError(w, http.StatusBadRequest, ErrMsgBadRequestInvalidRequestBody)
			return nil
		}

		s.respondWithError(w, http.StatusInternalServerError, ErrMsgInternalError)
		return fmt.Errorf("error while registering user: %w", err)
	}

	s.respondWithJSON(w, http.StatusOK, user)

	return nil
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) error {
	log.Println("[Server] Called POST /login")

	loginRequest := &models.LoginRequest{}
	if err := json.NewDecoder(r.Body).Decode(loginRequest); err != nil {
		s.respondWithError(w, http.StatusBadRequest, ErrMsgBadRequestInvalidRequestBody)
		return nil
	}

	token, err := s.userService.LoginUser(loginRequest.Email, loginRequest.Password)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			s.respondWithError(w, http.StatusUnauthorized, ErrMsgUnauthorizedInvalidCredentials)
			return nil
		}

		s.respondWithError(w, http.StatusInternalServerError, ErrMsgInternalError)
		return fmt.Errorf("error while registering user: %w", err)
	}

	s.respondWithJSON(w, http.StatusOK, models.LoginResponse{Token: token})

	return nil
}

func (s *Server) handleGetBooks(w http.ResponseWriter, r *http.Request) error {
	log.Println("[Server] Called GET /books")

	books, err := s.bookService.GetBooks()
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, ErrMsgInternalError)
		return fmt.Errorf("error while getting books: %w", err)
	}

	s.respondWithJSON(w, http.StatusOK, books)

	return nil
}

func (s *Server) handlePostBook(w http.ResponseWriter, r *http.Request) error {
	log.Println("[Server] Called POST /books")

	createBookRequest := &models.CreateBookRequest{}
	if err := json.NewDecoder(r.Body).Decode(createBookRequest); err != nil {
		s.respondWithError(w, http.StatusBadRequest, ErrMsgBadRequestInvalidRequestBody)
		return nil
	}

	book, err := s.bookService.AddBook(createBookRequest.Author, createBookRequest.Title)
	if err != nil {
		if errors.Is(err, services.ErrInvalidAuthor) {
			s.respondWithError(w, http.StatusBadRequest, ErrMsgBadRequestInvalidRequestBody)
			return nil
		}
		if errors.Is(err, services.ErrInvalidTitle) {
			s.respondWithError(w, http.StatusBadRequest, ErrMsgBadRequestInvalidRequestBody)
			return nil
		}

		s.respondWithError(w, http.StatusInternalServerError, ErrMsgInternalError)
		return fmt.Errorf("error while creating new book: %w", err)
	}

	s.respondWithJSON(w, http.StatusOK, book)

	return nil
}

func (s *Server) handleGetBookByID(w http.ResponseWriter, r *http.Request) error {
	log.Println("[Server] Called GET /books/{id}")

	idString := mux.Vars(r)["id"]
	defer r.Body.Close()

	id, err := strconv.Atoi(idString)
	if err != nil {
		s.respondWithError(w, http.StatusBadRequest, ErrMsgBadRequestInvalidBookID)
		return nil
	}

	book, err := s.bookService.GetBook(id)
	if err != nil {
		if errors.Is(err, services.ErrBookNotFound) {
			s.respondWithError(w, http.StatusNotFound, ErrMsgNotFound)
			return nil
		}

		s.respondWithError(w, http.StatusInternalServerError, ErrMsgInternalError)
		return fmt.Errorf("error while getting book: %w", err)
	}

	s.respondWithJSON(w, http.StatusOK, book)

	return nil
}

func (s *Server) handlePutBookByID(w http.ResponseWriter, r *http.Request) error {
	log.Println("[Server] Called PUT /books/{id}")

	idString := mux.Vars(r)["id"]
	defer r.Body.Close()

	id, err := strconv.Atoi(idString)
	if err != nil {
		s.respondWithError(w, http.StatusBadRequest, ErrMsgBadRequestInvalidBookID)
		return nil
	}

	book := &models.Book{}
	if err := json.NewDecoder(r.Body).Decode(book); err != nil {
		s.respondWithError(w, http.StatusBadRequest, ErrMsgBadRequestInvalidRequestBody)
		return nil
	}

	updatedBook, err := s.bookService.UpdateBook(id, book.Author, book.Title)
	if err != nil {
		if errors.Is(err, services.ErrBookNotFound) {
			s.respondWithError(w, http.StatusNotFound, ErrMsgNotFound)
			return nil
		}
		if errors.Is(err, services.ErrInvalidAuthorOrTitle) {
			s.respondWithError(w, http.StatusBadRequest, ErrMsgBadRequestInvalidRequestBody)
			return nil
		}

		s.respondWithError(w, http.StatusInternalServerError, ErrMsgInternalError)
		return fmt.Errorf("error while getting book: %w", err)
	}

	s.respondWithJSON(w, http.StatusOK, updatedBook)

	return nil
}

func (s *Server) handleDeleteBookByID(w http.ResponseWriter, r *http.Request) error {
	log.Println("[Server] Called DELETE /books/{id}")

	idString := mux.Vars(r)["id"]
	defer r.Body.Close()

	id, err := strconv.Atoi(idString)
	if err != nil {
		s.respondWithError(w, http.StatusBadRequest, ErrMsgBadRequestInvalidBookID)
		return nil
	}

	if err := s.bookService.DeleteBook(id); err != nil {
		if errors.Is(err, services.ErrBookNotFound) {
			s.respondWithError(w, http.StatusNotFound, ErrMsgNotFound)
			return nil
		}

		s.respondWithError(w, http.StatusInternalServerError, ErrMsgInternalError)
		return fmt.Errorf("error while getting book: %w", err)
	}

	s.respondWithJSON(w, http.StatusOK, nil)
	return nil
}

func (s *Server) validateJWT(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			s.respondWithError(w, http.StatusUnauthorized, ErrMsgUnauthorized)
			return
		}

		authHeaderParts := strings.Split(authHeader, " ")
		if len(authHeaderParts) != 2 || authHeaderParts[0] != "Bearer" {
			s.respondWithError(w, http.StatusUnauthorized, ErrMsgUnauthorized)
			return
		}

		tokenString := authHeaderParts[1]
		if err := s.userService.ValidateToken(tokenString); err != nil {
			if errors.Is(err, services.ErrExpiredToken) {
				s.respondWithError(w, http.StatusUnauthorized, ErrMsgUnauthorizedExpiredToken)
				return
			}
			if errors.Is(err, services.ErrInvalidToken) {
				s.respondWithError(w, http.StatusUnauthorized, ErrMsgUnauthorizedInvalidToken)
				return
			}

			s.respondWithError(w, http.StatusInternalServerError, ErrMsgInternalError)
			return
		}

		f(w, r)
	}
}

func (s *Server) respondWithError(w http.ResponseWriter, errCode int, errMessage string) {
	s.respondWithJSON(w, errCode, models.ErrorResponse{Error: errMessage})
}

func (s *Server) respondWithJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")

	response, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[Server] Error while marshaling JSON response: %s", err.Error())

		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(ErrMsgInternalError))

		return
	}

	w.WriteHeader(code)
	_, _ = w.Write(response)
}