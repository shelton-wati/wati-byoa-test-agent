package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/wati/wati-byoa-test-agent/internal/llm"
	"github.com/wati/wati-byoa-test-agent/internal/webhook"
)

type Scope struct {
	Conversation webhook.ConversationContext
}

type Tool struct {
	Name        string
	Description string
	Parameters  map[string]any
	Execute     func(ctx context.Context, scope Scope, args json.RawMessage) (string, error)
}

type Registry struct {
	tools []Tool
	byName map[string]Tool
}

func NewRegistry() *Registry {
	reg := &Registry{byName: make(map[string]Tool)}
	reg.register(currentTimeTool())
	reg.register(calculateTool())
	return reg
}

func (r *Registry) register(t Tool) {
	r.tools = append(r.tools, t)
	r.byName[t.Name] = t
}

func (r *Registry) Definitions() []llm.ToolDef {
	out := make([]llm.ToolDef, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, llm.ToolDef{
			Type: "function",
			Function: llm.FunctionDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		})
	}
	return out
}

func (r *Registry) Execute(ctx context.Context, scope Scope, name string, args json.RawMessage) (string, error) {
	t, ok := r.byName[name]
	if !ok {
		return "", fmt.Errorf("unknown tool %q", name)
	}
	return t.Execute(ctx, scope, args)
}

func currentTimeTool() Tool {
	return Tool{
		Name:        "get_current_time",
		Description: "Return the current UTC date and time.",
		Parameters: llm.ObjectSchema(nil, map[string]any{}),
		Execute: func(_ context.Context, _ Scope, _ json.RawMessage) (string, error) {
			return time.Now().UTC().Format(time.RFC3339), nil
		},
	}
}

func calculateTool() Tool {
	return Tool{
		Name:        "calculate",
		Description: "Evaluate a basic arithmetic expression with +, -, *, /, parentheses, and decimal numbers.",
		Parameters: llm.ObjectSchema([]string{"expression"}, map[string]any{
			"expression": map[string]any{
				"type":        "string",
				"description": "Arithmetic expression, e.g. (12 + 4) * 2",
			},
		}),
		Execute: func(_ context.Context, _ Scope, args json.RawMessage) (string, error) {
			var payload struct {
				Expression string `json:"expression"`
			}
			if err := json.Unmarshal(args, &payload); err != nil {
				return "", err
			}
			value, err := evalExpression(strings.TrimSpace(payload.Expression))
			if err != nil {
				return "", err
			}
			return strconv.FormatFloat(value, 'f', -1, 64), nil
		},
	}
}


func evalExpression(expr string) (float64, error) {
	if expr == "" {
		return 0, fmt.Errorf("expression is required")
	}
	for _, r := range expr {
		if !(unicode.IsDigit(r) || r == '+' || r == '-' || r == '*' || r == '/' || r == '.' || r == '(' || r == ')' || unicode.IsSpace(r)) {
			return 0, fmt.Errorf("unsupported character in expression")
		}
	}
	tokens, err := tokenize(expr)
	if err != nil {
		return 0, err
	}
	rpn, err := toRPN(tokens)
	if err != nil {
		return 0, err
	}
	return evalRPN(rpn)
}

type token struct {
	kind  string
	value float64
	text  string
}

func tokenize(expr string) ([]token, error) {
	var tokens []token
	i := 0
	for i < len(expr) {
		ch := rune(expr[i])
		if unicode.IsSpace(ch) {
			i++
			continue
		}
		switch ch {
		case '+', '-', '*', '/', '(', ')':
			tokens = append(tokens, token{kind: string(ch)})
			i++
			continue
		}
		if unicode.IsDigit(ch) || ch == '.' {
			j := i
			for j < len(expr) && (unicode.IsDigit(rune(expr[j])) || expr[j] == '.') {
				j++
			}
			value, err := strconv.ParseFloat(expr[i:j], 64)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, token{kind: "number", value: value})
			i = j
			continue
		}
		return nil, fmt.Errorf("invalid token at position %d", i)
	}
	return tokens, nil
}

func precedence(op string) int {
	switch op {
	case "+", "-":
		return 1
	case "*", "/":
		return 2
	default:
		return 0
	}
}

func toRPN(tokens []token) ([]token, error) {
	var output []token
	var stack []token
	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		switch t.kind {
		case "number":
			output = append(output, t)
		case "(":
			stack = append(stack, t)
		case ")":
			for len(stack) > 0 && stack[len(stack)-1].kind != "(" {
				output = append(output, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			if len(stack) == 0 {
				return nil, fmt.Errorf("mismatched parentheses")
			}
			stack = stack[:len(stack)-1]
		default:
			for len(stack) > 0 && stack[len(stack)-1].kind != "(" && precedence(stack[len(stack)-1].kind) >= precedence(t.kind) {
				output = append(output, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			stack = append(stack, t)
		}
	}
	for len(stack) > 0 {
		if stack[len(stack)-1].kind == "(" {
			return nil, fmt.Errorf("mismatched parentheses")
		}
		output = append(output, stack[len(stack)-1])
		stack = stack[:len(stack)-1]
	}
	return output, nil
}

func evalRPN(tokens []token) (float64, error) {
	var stack []float64
	for _, t := range tokens {
		if t.kind == "number" {
			stack = append(stack, t.value)
			continue
		}
		if len(stack) < 2 {
			return 0, fmt.Errorf("invalid expression")
		}
		b := stack[len(stack)-1]
		a := stack[len(stack)-2]
		stack = stack[:len(stack)-2]
		switch t.kind {
		case "+":
			stack = append(stack, a+b)
		case "-":
			stack = append(stack, a-b)
		case "*":
			stack = append(stack, a*b)
		case "/":
			if b == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			stack = append(stack, a/b)
		default:
			return 0, fmt.Errorf("unknown operator %q", t.kind)
		}
	}
	if len(stack) != 1 {
		return 0, fmt.Errorf("invalid expression")
	}
	if math.IsInf(stack[0], 0) || math.IsNaN(stack[0]) {
		return 0, fmt.Errorf("invalid numeric result")
	}
	return stack[0], nil
}
