# MCP Go Debugger Implementation Guide

This guide outlines best practices for implementing features in the MCP Go Debugger project.

## Development Workflow

When implementing a new feature or modifying existing code, follow these steps:

1. **Understand the Task**: Thoroughly review the relevant task description.
2. **Explore the Codebase**: Identify relevant files and understand existing patterns.
3. **Make Focused Changes**: Implement one coherent feature/fix at a time.
4. **Write Tests**: Add tests for the new functionality.
5. **Validate**: Run tests to ensure all tests pass.
6. **Commit**: Commit changes with descriptive messages.
7. **Repeat**: Only after successfully completing a change, move to the next task.

## Understanding the Codebase

Before making changes:

1. Explore the project structure
2. Identify key files and packages
3. Review existing tests for similar functionality

## Implementation Strategies

### Adding a New Feature

When adding a new feature:

1. **Review Similar Features**: Look at how similar features are implemented.
2. **Follow Conventions**: Match existing code style and patterns.
3. **Implement Core Logic**: Create the main functionality first.
4. **Add Error Handling**: Ensure proper error cases are handled.
5. **Update Interfaces**: Update any interfaces or API definitions.

### Modifying Existing Code

When modifying existing code:

1. **Locate Target Code**: Find all related code sections that need changes.
2. **Understand Dependencies**: Identify code that depends on what you're changing.
3. **Make Minimal Changes**: Change only what's necessary for the feature.
4. **Preserve Behavior**: Ensure existing functionality remains intact unless deliberately changing it.

## Test-Driven Development

Follow these testing principles:

1. **Write Tests First**: When possible, write tests before implementing features.
2. **Test Edge Cases**: Include tests for error conditions and edge cases.
3. **Ensure Coverage**: Aim for good test coverage of new functionality.
4. **Test Commands**: For each MCP command, test both success and failure paths.

## Running Tests

Run tests before committing:

```sh
go test ./...
```

For specific packages:

```sh
go test ./pkg/debugger
```

With verbose output:

```sh
go test -v ./...
```

## Git Workflow

For each feature implementation:

1. **Start Clean**: Begin with a clean working directory.
2. **Feature Branch**: Consider using feature branches for significant changes.
3. **Atomic Commits**: Make focused commits that implement one logical change.
4. **Descriptive Messages**: Use clear commit messages following this format:
   ```
   [Component] Brief description of change
   
   More detailed explanation if needed
   ```
5. **Verify Before Commit**: Always run tests before committing.

### Sample Commit Workflow

```sh
# Run tests to ensure everything passes before changes
go test ./...

# Make your changes

# Run tests again to verify changes
go test ./...

# Commit changes with descriptive message
git add .
git commit -m "[Debugger] Implement set_breakpoint command"
```

## Troubleshooting

If tests fail:

1. Review the test output carefully
2. Check for any side effects from your changes
3. Verify that you've updated all necessary parts of the codebase
4. Consider running tests with the `-v` flag for more details

## Documentation

Update documentation when:

1. Adding new commands or features
2. Changing existing behavior
3. Fixing non-obvious bugs

## API Documentation

Always use `go doc` instead of searching the internet when finding out Go APIs. This ensures that you're referencing the exact API version in your dependencies:

```sh
# View documentation for a package
go doc github.com/go-delve/delve/service

# View documentation for a specific type
go doc github.com/go-delve/delve/service.Config

# View documentation for a function or method
go doc github.com/go-delve/delve/service.NewServer

# For more detailed documentation including unexported items
go doc -all github.com/go-delve/delve/service
```

This approach ensures you're working with the precise API that's included in your project's dependencies rather than potentially outdated or incompatible online resources.

## Next Steps

After implementing a feature:

1. Consider if additional tests would be valuable
2. Look for opportunities to refactor or improve the code
3. Update the task status in Phase1-Tasks.md
4. Move on to the next related task 