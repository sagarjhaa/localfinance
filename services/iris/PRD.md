# Product Requirements Document: Iris

**Service Name:** Iris  
**Version:** 1.0  
**Date:** March 22, 2026  
**Document Type:** Product Requirements Document  
**Status:** Draft  

---

## 1. Executive Summary

Iris is the next-generation frontend service for LocalFinance, designed to replace the legacy Flask dashboard with a modern React-based user interface. Named after the Greek goddess of the rainbow and messenger, Iris serves as the colorful, user-friendly gateway between users and the LocalFinance ecosystem.

**Key Value Proposition:** Transform the user experience from a basic Flask dashboard to an intuitive, responsive, and feature-rich web application that streamlines financial data management.

---

## 2. Product Overview

### 2.1 Service Identity
- **Name:** Iris (Greek goddess of the rainbow/messenger)
- **Role:** Frontend UI service for LocalFinance platform
- **Port:** 3001
- **Technology Stack:** React + Node.js (Express)
- **Target Users:** LocalFinance platform users managing financial data

### 2.2 Mission Statement
Provide users with an elegant, efficient, and secure interface for uploading, processing, and analyzing their financial data through the LocalFinance ecosystem.

---

## 3. Goals & Objectives

### 3.1 Primary Goals
1. **Replace Legacy System:** Fully deprecate the existing Flask dashboard (app.py)
2. **Enhanced User Experience:** Deliver a modern, responsive UI with improved usability
3. **Scalability:** Build a foundation that can accommodate future feature expansion
4. **Security:** Implement robust authentication and session management
5. **Performance:** Achieve faster load times and better responsiveness than legacy system

### 3.2 Success Metrics
- **User Adoption:** 100% migration from legacy dashboard within 3 months
- **Performance:** Page load times under 2 seconds
- **User Satisfaction:** 90%+ positive feedback on new interface
- **Uptime:** 99.9% availability
- **Error Rate:** <1% failed file uploads

---

## 4. User Stories & Functional Requirements

### 4.1 File Upload Management

**User Story:** "As a user, I want to easily upload my bank statements so that I can analyze my financial data."

**Acceptance Criteria:**
- [ ] Drag & drop interface for file uploads
- [ ] Support for multiple file formats: CSV, PDF, Excel (.xlsx, .xls)
- [ ] File preview functionality before confirming upload
- [ ] Real-time progress bar during file processing
- [ ] Clear status messages for success/error states
- [ ] Ability to upload multiple files simultaneously
- [ ] File size validation (max 10MB per file)
- [ ] File format validation with user-friendly error messages

**Technical Requirements:**
- File validation on both client and server side
- Integration with Logos service (port 8003) for file processing
- Temporary file storage during processing
- Error handling and recovery mechanisms

### 4.2 Statistics Dashboard

**User Story:** "As a user, I want to see an overview of my financial data processing activity so that I can track my usage and success rates."

**Acceptance Criteria:**
- [ ] Display total files processed (all-time)
- [ ] Show total transactions extracted
- [ ] Calculate and display success rate percentage
- [ ] List recently processed files with timestamps
- [ ] Filter statistics by date range
- [ ] Export statistics as CSV/PDF
- [ ] Visual charts and graphs for data representation

**Technical Requirements:**
- Integration with Thesaurus service (port 8001) for data retrieval
- Real-time data updates
- Caching mechanism for frequently accessed statistics
- Responsive chart library integration

### 4.3 Authentication System (NEW)

**User Story:** "As a user, I want secure access to my financial data so that my information remains protected."

**Acceptance Criteria:**
- [ ] User registration with email verification
- [ ] Secure login/logout functionality
- [ ] Session management with automatic timeout
- [ ] Password reset capability
- [ ] Multi-factor authentication (MFA) support
- [ ] Remember me functionality
- [ ] Account lockout after failed attempts

**Technical Requirements:**
- JWT token-based authentication
- Password hashing using bcrypt
- Session storage and management
- Rate limiting for login attempts
- HTTPS enforcement
- Secure cookie handling

### 4.4 Settings & Configuration (NEW)

**User Story:** "As a user, I want to customize my experience and configure my preferences so that the application works best for my needs."

**Acceptance Criteria:**
- [ ] User profile management (name, email, preferences)
- [ ] Upload folder configuration
- [ ] Notification preferences (email, in-app)
- [ ] Theme selection (light/dark mode)
- [ ] Language preferences
- [ ] Data retention settings
- [ ] Export preferences (format, frequency)

**Technical Requirements:**
- User preferences stored in database
- Real-time preference updates
- Integration with notification systems
- Backup and restore of user settings

---

## 5. Technical Specifications

### 5.1 Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│     Client      │    │      Iris       │    │   LocalFinance  │
│   (Browser)     │◄──►│   Frontend      │◄──►│    Services     │
│                 │    │  (Port 3001)    │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │                       │
                                ▼                       ▼
                         ┌─────────────────┐    ┌─────────────────┐
                         │   Express.js    │    │   Thesaurus     │
                         │    Server       │    │  (Port 8001)    │
                         └─────────────────┘    └─────────────────┘
                                                ┌─────────────────┐
                                                │     Sophia      │
                                                │  (Port 8002)    │
                                                └─────────────────┘
                                                ┌─────────────────┐
                                                │      Logos      │
                                                │  (Port 8003)    │
                                                └─────────────────┘
```

### 5.2 Technology Stack

**Frontend:**
- React 18+ with TypeScript
- Material-UI or Chakra UI for components
- React Router for navigation
- Axios for API calls
- React Hook Form for form management
- Chart.js or Recharts for data visualization

**Backend:**
- Node.js with Express.js
- TypeScript for type safety
- JWT for authentication
- Multer for file uploads
- Helmet for security headers
- Morgan for logging
- Rate limiting middleware

### 5.3 Database Schema

```sql
-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    settings JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Sessions table
CREATE TABLE user_sessions (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- File processing history
CREATE TABLE file_uploads (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    filename VARCHAR(255) NOT NULL,
    file_type VARCHAR(50) NOT NULL,
    file_size INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL, -- 'processing', 'completed', 'failed'
    transactions_count INTEGER,
    error_message TEXT,
    uploaded_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP
);
```

---

## 6. API Integration

### 6.1 External Service Endpoints

**Thesaurus Service (Port 8001)** - Data Management
- `GET /api/statistics` - Retrieve user statistics
- `GET /api/transactions` - Fetch transaction data
- `GET /api/files` - Get file processing history

**Sophia Service (Port 8002)** - AI Insights
- `POST /api/analyze` - Generate AI insights from financial data
- `GET /api/insights/{userId}` - Retrieve saved insights

**Logos Service (Port 8003)** - File Processing
- `POST /api/upload` - Upload and process files
- `GET /api/status/{jobId}` - Check processing status
- `POST /api/validate` - Validate file format

### 6.2 Iris API Endpoints

**Authentication:**
- `POST /auth/register` - User registration
- `POST /auth/login` - User login
- `POST /auth/logout` - User logout
- `POST /auth/refresh` - Refresh JWT token
- `POST /auth/forgot-password` - Password reset request

**Files:**
- `POST /files/upload` - Upload files
- `GET /files/history` - Get upload history
- `DELETE /files/{id}` - Delete file record

**User Management:**
- `GET /user/profile` - Get user profile
- `PUT /user/profile` - Update user profile
- `GET /user/settings` - Get user settings
- `PUT /user/settings` - Update user settings

---

## 7. UI/UX Requirements

### 7.1 Design Principles
- **Simplicity:** Clean, uncluttered interface
- **Accessibility:** WCAG 2.1 AA compliance
- **Responsiveness:** Mobile-first design
- **Consistency:** Unified design language across all components
- **Performance:** Fast, responsive interactions

### 7.2 Key Interface Components

**Navigation:**
- Top navigation bar with logo, main menu, user profile
- Sidebar navigation for main sections
- Breadcrumb navigation for deep pages

**Dashboard:**
- Widget-based layout with customizable cards
- Quick action buttons for common tasks
- Recent activity feed

**File Upload:**
- Prominent drag-and-drop area
- File queue with progress indicators
- Clear error messaging with resolution suggestions

**Settings:**
- Tabbed interface for different setting categories
- Inline editing with immediate save
- Import/export functionality

### 7.3 Color Scheme & Branding
- Primary: Modern blue (#2563EB)
- Secondary: Warm orange (#F59E0B)
- Success: Green (#10B981)
- Warning: Amber (#F59E0B)
- Error: Red (#EF4444)
- Neutral grays for backgrounds and text

---

## 8. Security Requirements

### 8.1 Authentication & Authorization
- JWT-based authentication with refresh tokens
- Role-based access control (if multiple user types)
- Session timeout after 24 hours of inactivity
- Secure password requirements (8+ chars, mixed case, numbers, symbols)

### 8.2 Data Protection
- All API communications over HTTPS
- File uploads validated and sanitized
- Sensitive data encrypted at rest
- No financial data stored in frontend cache

### 8.3 Security Headers
- Content Security Policy (CSP)
- X-Frame-Options
- X-Content-Type-Options
- Strict-Transport-Security

---

## 9. Performance Requirements

### 9.1 Load Times
- Initial page load: <2 seconds
- Navigation between pages: <500ms
- File upload initialization: <1 second

### 9.2 Scalability
- Support for 100+ concurrent users
- Handle files up to 10MB efficiently
- Database queries optimized for <100ms response

### 9.3 Caching Strategy
- Static assets cached for 1 year
- API responses cached for 5 minutes
- User settings cached locally

---

## 10. Testing Strategy

### 10.1 Unit Testing
- React component testing with Jest and React Testing Library
- Node.js API testing with Jest and Supertest
- Code coverage target: 90%+

### 10.2 Integration Testing
- End-to-end testing with Playwright or Cypress
- API integration testing with external services
- File upload flow testing with various file types

### 10.3 Performance Testing
- Load testing for concurrent users
- File upload performance testing
- Database query performance testing

---

## 11. Deployment & DevOps

### 11.1 Environment Setup
- **Development:** Local development with hot reload
- **Staging:** Pre-production testing environment
- **Production:** Live environment with monitoring

### 11.2 CI/CD Pipeline
- Automated testing on pull requests
- Build and deployment automation
- Health checks and rollback capabilities

### 11.3 Monitoring
- Application performance monitoring
- Error tracking and alerting
- User analytics and usage metrics

---

## 12. Migration Strategy

### 12.1 Phase 1: Core Features (Month 1)
- Basic file upload functionality
- Statistics dashboard
- User authentication

### 12.2 Phase 2: Enhanced Features (Month 2)
- Advanced settings
- Improved UI/UX
- Performance optimizations

### 12.3 Phase 3: Full Migration (Month 3)
- Feature parity with legacy system
- User migration from Flask app
- Legacy system deprecation

---

## 13. Risk Assessment

### 13.1 Technical Risks
- **Risk:** Integration complexity with existing services
- **Mitigation:** Thorough API testing and documentation

- **Risk:** File upload performance issues
- **Mitigation:** Implement streaming uploads and progress tracking

### 13.2 User Adoption Risks
- **Risk:** User resistance to change
- **Mitigation:** Gradual rollout with user training

- **Risk:** Feature gaps compared to legacy system
- **Mitigation:** Comprehensive feature audit and user feedback

---

## 14. Future Enhancements

### 14.1 Phase 2 Features (Post-MVP)
- Real-time notifications
- Advanced data visualization
- Mobile application
- Batch processing capabilities
- Advanced search and filtering

### 14.2 Integration Opportunities
- Third-party bank API integrations
- Export to accounting software
- Automated categorization improvements
- Machine learning insights

---

## Appendix A: Glossary

- **Iris:** The frontend service being developed
- **Thesaurus:** LocalFinance data management service
- **Sophia:** LocalFinance AI insights service  
- **Logos:** LocalFinance file processing service
- **JWT:** JSON Web Token for authentication
- **MFA:** Multi-Factor Authentication

---

## Appendix B: References

- LocalFinance Architecture Documentation
- Flask Dashboard Legacy Code (app.py)
- UI/UX Design Guidelines
- Security Best Practices
- React Development Standards

---

**Document Approval:**

| Role | Name | Signature | Date |
|------|------|-----------|------|
| Product Manager | | | |
| Engineering Lead | | | |
| Design Lead | | | |
| Security Review | | | |

---

*This PRD serves as the foundational document for Iris development. All changes must be reviewed and approved through the established change management process.*