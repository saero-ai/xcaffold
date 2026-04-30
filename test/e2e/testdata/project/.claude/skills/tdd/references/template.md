## Test Template

Use this structure for all Go test functions:

```go
func TestCommand_Feature_Scenario(t *testing.T) {
    // Arrange: set up preconditions
    input := createTestFixture(t)

    // Act: execute the behavior under test
    result, err := functionUnderTest(input)

    // Assert: verify the outcome
    require.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

### Table-Driven Tests

For functions with multiple input/output combinations, use table-driven tests:

```go
func TestParse_VariousInputs(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {name: "valid input", input: "foo", expected: "FOO"},
        {name: "empty input", input: "", wantErr: true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Parse(tt.input)
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.expected, got)
        })
    }
}
```

### Benchmark Tests

For performance-sensitive code, add benchmarks:

```go
func BenchmarkCompile_LargeConfig(b *testing.B) {
    config := loadLargeFixture(b)
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        Compile(config, "", "claude", "")
    }
}
```
