# Example User Flow: Multi-Step E-Commerce Checkout

## 1. Flow Overview
- **Flow ID**: `checkout-e2e-v1`
- **Objective**: Guide an authenticated or guest user through cart validation, address input, payment method selection, and order confirmation.
- **Key Invariants**: Zero dead ends, preserve payment details across failures, automatic session resumption.

---

## 2. Mermaid Flowchart

```mermaid
flowchart TD
    Start([Cart: User Clicks Checkout]) --> AuthCheck{User Logged In?}
    
    %% Authentication fork
    AuthCheck -- Yes --> LoadSavedAddress[Load Default Shipping Address]
    AuthCheck -- No --> GuestOrLogin{Choose Checkout Type}
    
    GuestOrLogin -- Guest --> InputGuestAddress[Enter Guest Shipping Info]
    GuestOrLogin -- Login --> ModalLogin[Authenticate via Modal]
    ModalLogin --> LoadSavedAddress
    
    %% Address confirmation
    LoadSavedAddress --> VerifyAddress{Address Valid?}
    InputGuestAddress --> VerifyAddress
    
    VerifyAddress -- Yes --> PaymentSelection[Select Payment Method]
    VerifyAddress -- No --> FixAddress[Display Address Validation Warnings]
    FixAddress --> InputGuestAddress
    
    %% Payment step
    PaymentSelection --> ProcessPayment[/Execute Payment Gateway/]
    
    ProcessPayment --> PaymentOutcome{Payment Succeeded?}
    
    %% Error recovery loop
    PaymentOutcome -- No (Decline/Timeout) --> PaymentError[Show Reason & Recovery Options]
    PaymentError --> PaymentSelection
    
    %% Success terminal
    PaymentOutcome -- Yes --> OrderSuccess([Order Confirmation & Receipt])
```

---

## 3. Transition Matrix

| Step ID | Current State | Trigger | Outcome | Fallback / Error Recovery |
| :--- | :--- | :--- | :--- | :--- |
| **01** | `Cart` | Click "Proceed to Checkout" | Evaluates auth state | N/A |
| **02** | `GuestOrLogin` | Selects "Guest Checkout" | Prompts for email & shipping | Retain cart items |
| **03** | `ShippingAddress` | Clicks "Continue to Payment" | Validates postal code & street | Inline field errors; focus moved |
| **04** | `Payment` | Submits payment details | Gateway processing | Keep address, allow alternate card |
| **05** | `OrderSuccess` | HTTP 200 from backend | Shows confirmation order # | Resend confirmation button |
