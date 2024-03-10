# Market Backend

## Description
This project is a backend for a market application, providing APIs for managing products, user registration, and cart management.

## Installation
1. Clone the repository.
2. Install Go and set up the Go environment.
3. Install the required dependencies using `go get`.
4. Set up a PostgreSQL database and update the connection details in the main.go file.
5. Run the application using `go run main.go`.

## Usage
- Use the provided APIs to manage products, register users, and manage the cart.
- API endpoints:
  - /products: GET (get all products), POST (create a new product)
  - /cart: GET (get the user's cart)
  - /cart/add: POST (add a product to the cart)
  - /register: POST (register a new user)
  - /login: POST (login)
  - /reset-database: POST (reset DB)
## Contributing
Feel free to submit issues and pull requests.

## License
This project is licensed under the MIT License
