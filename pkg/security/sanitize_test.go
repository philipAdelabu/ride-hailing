package security

import (
	"strings"
	"testing"
)

// TestSanitizeString tests the SanitizeString function
func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"simple string", "hello world", "hello world"},
		{"whitespace trim", "  hello world  ", "hello world"},
		{"null bytes removed", "hello\x00world", "helloworld"},
		{"multiple null bytes", "\x00test\x00input\x00", "testinput"},
		{"preserves newlines", "hello\nworld", "hello\nworld"},
		{"preserves tabs", "hello\tworld", "hello\tworld"},
		{"removes control chars", "hello\x01\x02\x03world", "helloworld"},
		{"unicode preserved", "hello 世界", "hello 世界"},
		{"emoji preserved", "hello 👋", "hello 👋"},
		{"mixed content", "  hello\x00\x01world  ", "helloworld"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeString(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeString(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSanitizeHTML tests the SanitizeHTML function
func TestSanitizeHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"simple text", "hello world", "hello world"},
		{"script tag", "<script>alert('xss')</script>", "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"},
		{"html entities", "<div>test</div>", "&lt;div&gt;test&lt;/div&gt;"},
		{"ampersand", "foo & bar", "foo &amp; bar"},
		{"quotes", `"quoted" text`, "&#34;quoted&#34; text"},
		{"single quotes", "'quoted'", "&#39;quoted&#39;"},
		{"angle brackets", "1 < 2 > 0", "1 &lt; 2 &gt; 0"},
		{"mixed special chars", "<a href=\"test\">link</a>", "&lt;a href=&#34;test&#34;&gt;link&lt;/a&gt;"},
		{"nested tags", "<div><span>text</span></div>", "&lt;div&gt;&lt;span&gt;text&lt;/span&gt;&lt;/div&gt;"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeHTML(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeHTML(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSanitizeForSQL tests the SanitizeForSQL function
func TestSanitizeForSQL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string // check that dangerous patterns are removed
	}{
		{"empty string", "", ""},
		{"simple text", "hello world", "hello world"},
		{"union select attack", "1 UNION SELECT * FROM users", ""},
		{"insert attack", "INSERT INTO users VALUES(1)", ""},
		{"delete attack", "DELETE FROM users WHERE 1=1", ""},
		{"drop table attack", "DROP TABLE users", ""},
		{"update set attack", "UPDATE users SET admin=1", ""},
		{"exec attack", "exec(malicious_code)", ""},
		{"execute attack", "EXECUTE sp_executesql", ""},
		{"case insensitive union", "union select", ""},
		{"mixed case attack", "uNiOn SeLeCt * FROM users", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeForSQL(tt.input)
			if tt.contains != "" && !strings.Contains(result, tt.contains) {
				t.Errorf("SanitizeForSQL(%q) should contain %q, got %q", tt.input, tt.contains, result)
			}
			// Ensure dangerous patterns are removed
			if strings.Contains(strings.ToLower(result), "union select") {
				t.Errorf("SanitizeForSQL(%q) should not contain 'union select', got %q", tt.input, result)
			}
			if strings.Contains(strings.ToLower(result), "insert into") {
				t.Errorf("SanitizeForSQL(%q) should not contain 'insert into', got %q", tt.input, result)
			}
		})
	}
}

// TestSanitizeForXSS tests the SanitizeForXSS function
func TestSanitizeForXSS(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"simple text", "hello world"},
		{"script tag", "<script>alert('xss')</script>"},
		{"script with attributes", "<script src='evil.js'></script>"},
		{"iframe attack", "<iframe src='evil.com'></iframe>"},
		{"onclick event", "<div onclick='alert(1)'>click</div>"},
		{"onload event", "<img onload='alert(1)'>"},
		{"onerror event", "<img onerror='alert(1)'>"},
		{"javascript protocol", "javascript:alert(1)"},
		{"embed tag", "<embed src='evil.swf'>"},
		{"object tag", "<object data='evil'>"},
		{"mixed attacks", "<script>alert(1)</script><iframe></iframe>"},
		{"uppercase script", "<SCRIPT>alert(1)</SCRIPT>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeForXSS(tt.input)
			// Verify dangerous patterns are removed
			if strings.Contains(strings.ToLower(result), "<script") {
				t.Errorf("SanitizeForXSS(%q) should not contain '<script', got %q", tt.input, result)
			}
			if strings.Contains(strings.ToLower(result), "<iframe") {
				t.Errorf("SanitizeForXSS(%q) should not contain '<iframe', got %q", tt.input, result)
			}
			if strings.Contains(strings.ToLower(result), "javascript:") {
				t.Errorf("SanitizeForXSS(%q) should not contain 'javascript:', got %q", tt.input, result)
			}
		})
	}
}

// TestSanitizeEmail tests the SanitizeEmail function
func TestSanitizeEmail(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"simple email", "test@example.com", "test@example.com"},
		{"uppercase email", "TEST@EXAMPLE.COM", "test@example.com"},
		{"mixed case", "Test.User@Example.COM", "test.user@example.com"},
		{"with plus sign", "test+tag@example.com", "test+tag@example.com"},
		{"with numbers", "user123@example.com", "user123@example.com"},
		{"with dots", "first.last@example.com", "first.last@example.com"},
		{"with underscore", "first_last@example.com", "first_last@example.com"},
		{"with hyphen", "first-last@example.com", "first-last@example.com"},
		{"whitespace trimmed", "  test@example.com  ", "test@example.com"},
		{"removes special chars", "test!#$@example.com", "test@example.com"},
		{"removes unicode", "test@exämple.com", "test@exmple.com"},
		{"removes spaces", "test @ example.com", "test@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeEmail(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeEmail(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSanitizePhone tests the SanitizePhone function
func TestSanitizePhone(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"digits only", "1234567890", "1234567890"},
		{"with plus", "+1234567890", "+1234567890"},
		{"formatted US", "(555) 123-4567", "5551234567"},
		{"with dashes", "555-123-4567", "5551234567"},
		{"with spaces", "555 123 4567", "5551234567"},
		{"international format", "+1 (555) 123-4567", "+15551234567"},
		{"with dots", "555.123.4567", "5551234567"},
		{"with letters", "555abc1234", "5551234"},
		{"mixed format", "+1-555-123-4567", "+15551234567"},
		{"extra plus signs", "++1234567890", "++1234567890"},
		{"leading zeros", "01onal234567890", "01234567890"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizePhone(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizePhone(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSanitizeAlphanumeric tests the SanitizeAlphanumeric function
func TestSanitizeAlphanumeric(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"alphanumeric only", "abc123", "abc123"},
		{"with spaces", "abc 123", "abc123"},
		{"with special chars", "abc!@#123", "abc123"},
		{"with unicode letters", "abc日本語123", "abc日本語123"},
		{"only special chars", "!@#$%^&*()", ""},
		{"mixed content", "Hello, World! 123", "HelloWorld123"},
		{"with newlines", "Hello\nWorld", "HelloWorld"},
		{"with tabs", "Hello\tWorld", "HelloWorld"},
		{"uppercase", "ABC123", "ABC123"},
		{"with underscores", "hello_world_123", "helloworld123"},
		{"with hyphens", "hello-world-123", "helloworld123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeAlphanumeric(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeAlphanumeric(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSanitizeFilename tests the SanitizeFilename function
func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"simple filename", "file.txt", "file.txt"},
		{"with path separator", "/path/to/file.txt", "pathtofile.txt"},
		{"with backslash", "path\\to\\file.txt", "pathtofile.txt"},
		{"directory traversal", "../../../etc/passwd", "etcpasswd"},
		{"double dots", "file..txt", "filetxt"},
		{"with spaces", "my file.txt", "my_file.txt"},
		{"with special chars", "file@#$.txt", "file___.txt"},
		{"long filename", strings.Repeat("a", 300), strings.Repeat("a", 255)},
		{"unicode chars", "файл.txt", "____.txt"},
		{"mixed traversal", "..\\..\\file.txt", "file.txt"},
		{"hyphen allowed", "my-file.txt", "my-file.txt"},
		{"underscore allowed", "my_file.txt", "my_file.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSanitizeURL tests the SanitizeURL function
func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"http url", "http://example.com", "http://example.com"},
		{"https url", "https://example.com", "https://example.com"},
		{"with path", "https://example.com/path/to/page", "https://example.com/path/to/page"},
		{"with query", "https://example.com?foo=bar", "https://example.com?foo=bar"},
		{"javascript protocol", "javascript:alert(1)", ""},
		{"javascript in path", "https://example.com/javascript:alert(1)", ""},
		{"ftp protocol", "ftp://example.com", ""},
		{"file protocol", "file:///etc/passwd", ""},
		{"data protocol", "data:text/html,<script>", ""},
		{"whitespace trimmed", "  https://example.com  ", "https://example.com"},
		{"uppercase javascript", "JAVASCRIPT:alert(1)", ""},
		{"mixed case", "JaVaScRiPt:alert(1)", ""},
		{"no protocol", "example.com", ""},
		{"relative path", "/path/to/page", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeURL(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeURL(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestStripHTMLTags tests the StripHTMLTags function
func TestStripHTMLTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"no tags", "hello world", "hello world"},
		{"single tag", "<p>hello</p>", "hello"},
		{"nested tags", "<div><p>hello</p></div>", "hello"},
		{"self-closing tags", "<br/>hello<hr/>", "hello"},
		{"multiple tags", "<h1>Title</h1><p>Paragraph</p>", "TitleParagraph"},
		{"with attributes", "<a href='url'>link</a>", "link"},
		{"script tag", "<script>alert(1)</script>", "alert(1)"},
		{"style tag", "<style>.class{}</style>", ".class{}"},
		{"comment", "<!-- comment -->hello", "hello"},
		{"malformed tag", "<p>hello</div>", "hello"},
		{"uppercase tags", "<DIV>hello</DIV>", "hello"},
		{"mixed content", "Hello <b>World</b>!", "Hello World!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripHTMLTags(tt.input)
			if result != tt.expected {
				t.Errorf("StripHTMLTags(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestStripNonAllowedHTMLTags tests the StripNonAllowedHTMLTags function
func TestStripNonAllowedHTMLTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"no tags", "hello world", "hello world"},
		{"allowed tag b", "<b>bold</b>", "<b>bold</b>"},
		{"allowed tag i", "<i>italic</i>", "<i>italic</i>"},
		{"allowed tag em", "<em>emphasis</em>", "<em>emphasis</em>"},
		{"allowed tag strong", "<strong>strong</strong>", "<strong>strong</strong>"},
		{"allowed tag p", "<p>paragraph</p>", "<p>paragraph</p>"},
		{"allowed tag br", "line<br>break", "line<br>break"},
		{"allowed tag span", "<span>text</span>", "<span>text</span>"},
		{"disallowed tag div", "<div>div content</div>", "div content"},
		{"disallowed tag script", "<script>alert(1)</script>", "alert(1)"},
		{"disallowed tag a", "<a href='url'>link</a>", "link"},
		{"strips attributes", "<b class='bold'>text</b>", "<b>text</b>"},
		{"mixed allowed and disallowed", "<div><b>bold</b></div>", "<b>bold</b>"},
		{"nested allowed", "<p><strong>bold</strong></p>", "<p><strong>bold</strong></p>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripNonAllowedHTMLTags(tt.input)
			if result != tt.expected {
				t.Errorf("StripNonAllowedHTMLTags(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestTruncateString tests the TruncateString function
func TestTruncateString(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		maxLength int
		expected  string
	}{
		{"empty string", "", 10, ""},
		{"shorter than max", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"longer than max", "hello world", 5, "hello"},
		{"zero max length", "hello", 0, ""},
		{"shorter unicode", "hello", 10, "hello"},
		{"single char", "a", 1, "a"},
		{"truncate to 1", "hello", 1, "h"},
		{"very long string", strings.Repeat("a", 1000), 100, strings.Repeat("a", 100)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateString(tt.input, tt.maxLength)
			if result != tt.expected {
				t.Errorf("TruncateString(%q, %d) = %q, want %q", tt.input, tt.maxLength, result, tt.expected)
			}
		})
	}
}

// TestNormalizeWhitespace tests the NormalizeWhitespace function
func TestNormalizeWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"single space", "hello world", "hello world"},
		{"multiple spaces", "hello    world", "hello world"},
		{"tabs", "hello\t\tworld", "hello world"},
		{"newlines", "hello\n\nworld", "hello world"},
		{"mixed whitespace", "hello  \t\n  world", "hello world"},
		{"leading whitespace", "   hello world", "hello world"},
		{"trailing whitespace", "hello world   ", "hello world"},
		{"only whitespace", "     ", ""},
		{"single word", "hello", "hello"},
		{"complex mixed", "  hello  \t world  \n foo  ", "hello world foo"},
		{"carriage return", "hello\r\nworld", "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeWhitespace(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeWhitespace(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestContainsSQLInjection tests the ContainsSQLInjection function
func TestContainsSQLInjection(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"empty string", "", false},
		{"normal text", "hello world", false},
		{"select statement", "SELECT * FROM users", false},
		{"union select", "UNION SELECT * FROM users", true},
		{"insert into", "INSERT INTO users VALUES(1)", true},
		{"delete from", "DELETE FROM users", true},
		{"drop table", "DROP TABLE users", true},
		{"update set", "UPDATE users SET admin=1", true},
		{"exec function", "exec(code)", true},
		{"execute function", "execute(code)", true},
		{"lowercase attack", "union select * from users", true},
		{"mixed case", "uNiOn SeLeCt", true},
		{"with whitespace", "UNION    SELECT", true},
		{"javascript in script", "script>alert(1)", true},
		{"javascript protocol", "javascript:alert(1)", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsSQLInjection(tt.input)
			if result != tt.expected {
				t.Errorf("ContainsSQLInjection(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestContainsXSS tests the ContainsXSS function
func TestContainsXSS(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"empty string", "", false},
		{"normal text", "hello world", false},
		{"script tag", "<script>alert(1)</script>", true},
		{"iframe tag", "<iframe src='evil'></iframe>", true},
		{"onclick event", "onclick='alert(1)'", true},
		{"onload event", "onload='alert(1)'", true},
		{"onerror event", "onerror='alert(1)'", true},
		{"onmouseover event", "onmouseover='alert(1)'", true},
		{"javascript protocol", "javascript:alert(1)", true},
		{"embed tag", "<embed src='evil'>", true},
		{"object tag", "<object data='evil'>", true},
		{"uppercase script", "<SCRIPT>alert(1)</SCRIPT>", true},
		{"mixed case", "<ScRiPt>alert(1)</sCrIpT>", true},
		{"inline script", "<script>alert(1)</script>text", true},
		{"harmless angle brackets", "1 < 2 > 0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsXSS(tt.input)
			if result != tt.expected {
				t.Errorf("ContainsXSS(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSanitizeInput tests the SanitizeInput function
func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		maxLength int
	}{
		{"empty string", "", 100},
		{"simple text", "hello world", 100},
		{"with xss", "<script>alert(1)</script>hello", 100},
		{"with sql injection", "hello UNION SELECT * FROM users", 100},
		{"with whitespace", "  hello   world  ", 100},
		{"long text truncated", strings.Repeat("a", 200), 100},
		{"null bytes", "hello\x00world", 100},
		{"control chars", "hello\x01\x02world", 100},
		{"combined attacks", "<script>UNION SELECT</script>   test   ", 100},
		{"zero max length", "hello", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeInput(tt.input, tt.maxLength)
			// Verify no XSS patterns
			if ContainsXSS(result) {
				t.Errorf("SanitizeInput(%q, %d) still contains XSS patterns", tt.input, tt.maxLength)
			}
			// Verify no SQL injection patterns
			if ContainsSQLInjection(result) {
				t.Errorf("SanitizeInput(%q, %d) still contains SQL injection patterns", tt.input, tt.maxLength)
			}
			// Verify length
			if tt.maxLength > 0 && len(result) > tt.maxLength {
				t.Errorf("SanitizeInput(%q, %d) length %d exceeds max length", tt.input, tt.maxLength, len(result))
			}
		})
	}
}

// TestUserInput_Sanitize tests the UserInput.Sanitize method
func TestUserInput_Sanitize(t *testing.T) {
	tests := []struct {
		name     string
		input    UserInput
		expected UserInput
	}{
		{
			name: "normal input",
			input: UserInput{
				Email:       "TEST@EXAMPLE.COM",
				Phone:       "(555) 123-4567",
				Name:        "John Doe",
				Description: "A simple description",
				URL:         "https://example.com",
			},
			expected: UserInput{
				Email:       "test@example.com",
				Phone:       "5551234567",
				Name:        "John Doe",
				Description: "A simple description",
				URL:         "https://example.com",
			},
		},
		{
			name: "xss in name",
			input: UserInput{
				Email:       "test@example.com",
				Phone:       "1234567890",
				Name:        "<script>alert(1)</script>John",
				Description: "Description",
				URL:         "https://example.com",
			},
			expected: UserInput{
				Email: "test@example.com",
				Phone: "1234567890",
				URL:   "https://example.com",
			},
		},
		{
			name: "javascript url",
			input: UserInput{
				Email:       "test@example.com",
				Phone:       "1234567890",
				Name:        "John",
				Description: "Desc",
				URL:         "javascript:alert(1)",
			},
			expected: UserInput{
				Email: "test@example.com",
				Phone: "1234567890",
				Name:  "John",
				URL:   "",
			},
		},
		{
			name: "empty input",
			input: UserInput{
				Email:       "",
				Phone:       "",
				Name:        "",
				Description: "",
				URL:         "",
			},
			expected: UserInput{
				Email:       "",
				Phone:       "",
				Name:        "",
				Description: "",
				URL:         "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.input
			input.Sanitize()

			if input.Email != tt.expected.Email {
				t.Errorf("Email: got %q, want %q", input.Email, tt.expected.Email)
			}
			if input.Phone != tt.expected.Phone {
				t.Errorf("Phone: got %q, want %q", input.Phone, tt.expected.Phone)
			}
			if input.URL != tt.expected.URL {
				t.Errorf("URL: got %q, want %q", input.URL, tt.expected.URL)
			}
		})
	}
}

// TestRemoveControlCharacters tests the removeControlCharacters function
func TestRemoveControlCharacters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"no control chars", "hello world", "hello world"},
		{"with newline", "hello\nworld", "hello\nworld"},
		{"with tab", "hello\tworld", "hello\tworld"},
		{"with bell", "hello\aworld", "helloworld"},
		{"with backspace", "hello\bworld", "helloworld"},
		{"with form feed", "hello\fworld", "helloworld"},
		{"with carriage return", "hello\rworld", "helloworld"},
		{"multiple control chars", "\x00\x01\x02hello\x03\x04", "hello"},
		{"printable range", " !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~", " !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeControlCharacters(tt.input)
			if result != tt.expected {
				t.Errorf("removeControlCharacters(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Benchmark tests
func BenchmarkSanitizeString(b *testing.B) {
	input := "  hello\x00world\x01test  "
	for i := 0; i < b.N; i++ {
		SanitizeString(input)
	}
}

func BenchmarkSanitizeForXSS(b *testing.B) {
	input := "<script>alert(1)</script>hello world"
	for i := 0; i < b.N; i++ {
		SanitizeForXSS(input)
	}
}

func BenchmarkSanitizeForSQL(b *testing.B) {
	input := "UNION SELECT * FROM users WHERE 1=1"
	for i := 0; i < b.N; i++ {
		SanitizeForSQL(input)
	}
}

func BenchmarkSanitizeEmail(b *testing.B) {
	input := "TEST@EXAMPLE.COM"
	for i := 0; i < b.N; i++ {
		SanitizeEmail(input)
	}
}

func BenchmarkSanitizeInput(b *testing.B) {
	input := "<script>UNION SELECT</script>   hello world   "
	for i := 0; i < b.N; i++ {
		SanitizeInput(input, 100)
	}
}
