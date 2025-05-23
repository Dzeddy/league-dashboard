# Contributing to League Dashboard

Thank you for your interest in contributing to League Dashboard! This document provides guidelines and information for contributors.

## 🤝 How to Contribute

### Reporting Issues

If you find a bug or have a feature request:

1. **Search existing issues** to avoid duplicates
2. **Use issue templates** when available
3. **Provide detailed information** including:
   - Steps to reproduce (for bugs)
   - Expected vs actual behavior
   - Environment details (OS, Node.js version, Go version)
   - Screenshots if applicable

### Feature Requests

We welcome feature requests! Please:

1. **Check existing issues** to see if someone else has suggested it
2. **Describe the problem** you're trying to solve
3. **Explain your proposed solution** with as much detail as possible
4. **Consider the scope** - is this something that would benefit most users?

## 🛠️ Development Process

### Setting Up Development Environment

1. **Fork and clone** the repository
   ```bash
   git clone https://github.com/yourusername/league_dashboard.git
   cd league_dashboard
   ```

2. **Install dependencies**
   ```bash
   npm run install:all
   ```

3. **Set up environment variables**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

4. **Start development servers**
   ```bash
   npm run dev
   ```

### Making Changes

1. **Create a feature branch** from `main`
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following our coding standards

3. **Test your changes**
   ```bash
   # Frontend tests
   npm run test
   
   # Backend tests
   npm run test:backend
   ```

4. **Commit your changes** with a clear message
   ```bash
   git commit -m "feat: add new player statistics visualization"
   ```

5. **Push to your fork** and create a pull request

## 📝 Coding Standards

### Frontend (React/TypeScript)

- **Use TypeScript** for all new code
- **Follow React best practices**
- **Use functional components** with hooks
- **Maintain consistent naming**:
  - Components: `PascalCase`
  - Functions/variables: `camelCase`
  - Constants: `UPPER_SNAKE_CASE`
- **Add proper types** for all props and state
- **Write meaningful comments** for complex logic

### Backend (Go)

- **Follow Go conventions** and best practices
- **Use meaningful package names**
- **Write clear function documentation**
- **Handle errors appropriately**
- **Use consistent naming**:
  - Exported functions/types: `PascalCase`
  - Unexported functions/variables: `camelCase`
- **Add tests** for new functionality

### General Guidelines

- **Write clear, self-documenting code**
- **Keep functions small and focused**
- **Use meaningful variable and function names**
- **Add comments for complex business logic**
- **Follow existing code patterns**

## 🧪 Testing

### Frontend Testing

- Write unit tests for components using React Testing Library
- Test user interactions and edge cases
- Maintain good test coverage for new features

### Backend Testing

- Write unit tests for handlers and business logic
- Test error scenarios and edge cases
- Use table-driven tests where appropriate

### Running Tests

```bash
# All tests
npm test

# Frontend only
cd frontend && npm test

# Backend only
npm run test:backend
```

## 📚 Documentation

- Update README.md if you add new features or change setup
- Add inline documentation for complex functions
- Update API documentation for new endpoints
- Include examples in your documentation

## 🔄 Pull Request Process

1. **Fill out the PR template** completely
2. **Ensure all tests pass**
3. **Update documentation** as needed
4. **Keep PRs focused** - one feature/fix per PR
5. **Respond to feedback** in a timely manner
6. **Squash commits** if requested

### PR Title Format

Use conventional commit format:
- `feat: add new feature`
- `fix: resolve bug with player search`
- `docs: update API documentation`
- `refactor: improve code structure`
- `test: add unit tests for handlers`

## 🎯 Project Goals

When contributing, keep these project goals in mind:

- **User Experience**: Prioritize intuitive and responsive design
- **Performance**: Optimize for fast loading and smooth interactions
- **Reliability**: Ensure robust error handling and data validation
- **Maintainability**: Write clean, well-documented code
- **Scalability**: Consider future growth and extensibility

## 🚫 What We Don't Accept

- Breaking changes without discussion
- Code that doesn't follow our standards
- Features that significantly increase complexity without clear benefit
- Changes that break existing functionality
- Code without appropriate tests

## 💬 Getting Help

- **Join our discussions** for questions and ideas
- **Check existing issues** for common problems
- **Ask questions** in your PR if you need clarification
- **Reach out to maintainers** for significant changes

## 🎉 Recognition

Contributors will be recognized in:
- The project's README.md
- Release notes for significant contributions
- GitHub's contributor graph

Thank you for helping make League Dashboard better for everyone!

## 📄 Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](https://www.contributor-covenant.org/). By participating, you agree to uphold this code.

## 📜 License

By contributing to this project, you agree that your contributions will be licensed under the MIT License. 