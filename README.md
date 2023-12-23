# Gophermart API Documentation

## Overview

Welcome to the Gophermart API, an HTTP interface designed for interacting with the Gofermart loyalty system. This service allows users to manage orders, credit/debit their wallets, and utilize loyalty points accrued from purchases in the Gofermart online store.

### Key Features:

- **User Registration:** New users can register to the system, creating a unique login.
- **Authentication and Authorization:** Secure login process and access control.
- **Order Management:** Users can submit and track their purchase orders.
- **Loyalty Points:** Earn, track, and redeem loyalty points for purchases.
- **Account Management:** View and manage account balance and transactions.

## Abstract Interaction Scheme

1. **User Registration:** Users sign up to the Gophermart loyalty system.
2. **Making a Purchase:** Users buy items from the Gofermart online store.
3. **Loyalty Points Calculation:** The system processes the order for loyalty points.
4. **Order Submission:** Users submit their order number to the loyalty system.
5. **Order Verification & Points Accrual:** The system verifies the order and credits loyalty points.
6. **Redemption:** Users redeem points for discounts on future purchases.

## API Endpoints

### User Management

- **POST /api/user/register:** Register a new user.
- **POST /api/user/login:** Authenticate a user and retrieve a token.

### Order Handling

- **GET /api/user/orders:** Retrieve a list of submitted orders.
- **POST /api/user/orders:** Submit a new order number.

### Balance & Transactions

- **GET /api/user/balance:** View current balance and total loyalty points.
- **POST /api/user/balance/withdraw:** Withdraw points for a new order.
- **GET /api/user/withdrawals:** Retrieve information about fund withdrawals.

## Security

- **ApiKeyAuth:** Secure API access with bearer token authorization.

## Error Handling

Standard HTTP status codes are used to indicate the success or failure of an API request. In case of an error, a detailed message will be returned to aid in debugging.

## External Documentation

- **Swagger:** Explore the full API specifications and interact with the API directly through the Swagger UI.

## Contact Information

For further inquiries or assistance, please contact Nikita Aleksandrov at nik29200018@gmail.com.

## License

This API is provided under the Apache 2.0 license. For more details, visit the [license page](http://www.apache.org/licenses/LICENSE-2.0.html).

---
