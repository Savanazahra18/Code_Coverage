package controllers

import (
	"net/http"

	"github.com/gieart87/gotoko/app/consts"
	"github.com/gieart87/gotoko/app/middlewares"
	"github.com/gorilla/mux"
)

func (server *Server) initializeRoutes() {
	server.Router = mux.NewRouter()
	server.Router.HandleFunc("/", server.Home).Methods("GET")

	server.Router.HandleFunc("/login", server.Login).Methods("GET")
	server.Router.HandleFunc("/login", server.DoLogin).Methods("POST")
	server.Router.HandleFunc("/register", server.Register).Methods("GET")
	server.Router.HandleFunc("/register", server.DoRegister).Methods("POST")
	server.Router.HandleFunc("/logout", server.Logout).Methods("GET")

	server.Router.HandleFunc("/products/search", server.SearchProducts).Methods("GET")
	server.Router.HandleFunc("/products", middlewares.AuthMiddleware(server.Products)).Methods("GET")
	server.Router.HandleFunc("/products/{slug}", server.GetProductBySlug).Methods("GET")

	server.Router.HandleFunc("/carts", server.GetCart).Methods("GET")
	server.Router.HandleFunc("/carts", server.AddItemToCart).Methods("POST")
	server.Router.HandleFunc("/carts/update", server.UpdateCart).Methods("POST")
	server.Router.HandleFunc("/carts/remove/{id}", server.RemoveItemByID).Methods("GET")
	server.Router.HandleFunc("/carts/shipping", server.CalculateShipping).Methods("POST")
	server.Router.HandleFunc("/orders/checkout", middlewares.AuthMiddleware(server.Checkout)).Methods("POST")
	server.Router.HandleFunc("/orders/{id}", middlewares.AuthMiddleware(server.ShowOrder)).Methods("GET")
	server.Router.HandleFunc("/payments/midtrans", server.Midtrans).Methods("POST")
	server.Router.HandleFunc("/admin/dashboard", middlewares.AuthMiddleware(middlewares.RoleMiddleware(server.AdminDashboard, server.DB, consts.RoleAdmin, consts.RoleOperator))).Methods("GET")
	server.Router.HandleFunc("/admin/products", server.AdminProducts).Methods("GET")
	server.Router.HandleFunc("/admin/products/create", server.CreateProductPage).Methods("GET")
	server.Router.HandleFunc("/admin/products/store", server.StoreProduct).Methods("POST")

	server.Router.HandleFunc("/admin/products/edit/{id}", server.EditProductPage).Methods("GET")
	server.Router.HandleFunc("/admin/products/update/{id}", server.UpdateProduct).Methods("POST")
	server.Router.HandleFunc("/admin/products/delete/{id}", server.DeleteProduct).Methods("POST")
	server.Router.HandleFunc("/admin/order-dashboard", server.OrderDashboard).Methods("GET")
	server.Router.HandleFunc("/admin/customers", server.ListCustomers).Methods("GET")
	server.Router.HandleFunc("/admin/order-items", server.ListOrderItems).Methods("GET")
	server.Router.HandleFunc("/admin/orders", server.ListOrders).Methods("GET")
	

staticDir := http.Dir("./assets")
staticHandler := http.StripPrefix("/assets/", http.FileServer(staticDir))

server.Router.PathPrefix("/assets/").Handler(staticHandler).Methods("GET")

}