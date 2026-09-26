package billingexpr

import (
	"fmt"
	"math"

	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
)

// ImageUnitPrice accepts only a single unconditional, fixed price multiplied
// by image_count. Other valid expressions may change price with request or usage.
func ImageUnitPrice(expression string) (float64, bool) {
	version, body := ParseExprVersion(expression)
	if version != DefaultExprVersion {
		return 0, false
	}
	tree, err := parser.Parse(body)
	if err != nil {
		return 0, false
	}
	multiplier, ok := tree.Node.(*ast.BinaryNode)
	if !ok || multiplier.Operator != "*" {
		return 0, false
	}
	priceNode := multiplier.Left
	countNode := multiplier.Right
	if count, valid := priceNode.(*ast.IdentifierNode); valid && count.Value == "image_count" {
		priceNode, countNode = countNode, priceNode
	}
	count, valid := countNode.(*ast.IdentifierNode)
	if !valid || count.Value != "image_count" {
		return 0, false
	}
	tier, valid := priceNode.(*ast.CallNode)
	if !valid || len(tier.Arguments) != 2 {
		return 0, false
	}
	callee, valid := tier.Callee.(*ast.IdentifierNode)
	if !valid || callee.Value != "tier" {
		return 0, false
	}
	if _, valid = tier.Arguments[0].(*ast.StringNode); !valid {
		return 0, false
	}
	fixed, valid := tier.Arguments[1].(*ast.CallNode)
	if !valid || len(fixed.Arguments) != 1 {
		return 0, false
	}
	callee, valid = fixed.Callee.(*ast.IdentifierNode)
	if !valid || callee.Value != "fixed" {
		return 0, false
	}
	amount, literal := requestRuleNumber(fixed.Arguments[0])
	return amount, literal && amount >= 0 && !math.IsNaN(amount) && !math.IsInf(amount*1_000_000, 0)
}

// UsesFixedPricing includes unselected branches, even when compilation later
// optimizes them away. Hosts use it to reject unsupported billing entrances.
func UsesFixedPricing(expression string) bool {
	return UsesFixedPricingByHash(expression, ExprHashString(expression))
}

// UsesFixedPricingByHash avoids hashing an expression already in a snapshot.
func UsesFixedPricingByHash(expression, hash string) bool {
	entry, err := compileEntryFromCacheByHash(expression, hash)
	return err == nil && entry.fixedPricing
}

func containsPricingMarker(node ast.Node) bool {
	return ast.Find(node, func(part ast.Node) bool {
		identifier, ok := part.(*ast.IdentifierNode)
		return ok && (identifier.Value == "tier" || identifier.Value == "fixed")
	}) != nil
}

func isRequestPriceMultiplier(node ast.Node) bool {
	if identifier, ok := node.(*ast.IdentifierNode); ok && identifier.Value == "image_count" {
		return true
	}
	conditional, ok := node.(*ast.ConditionalNode)
	if !ok || !usesRequestProbe(conditional.Cond) || containsPricingMarker(conditional.Cond) {
		return false
	}
	multiplier, multiplierOK := requestRuleNumber(conditional.Exp1)
	fallback, fallbackOK := requestRuleNumber(conditional.Exp2)
	return multiplierOK && fallbackOK && fallback == 1 && multiplier >= 0 && !math.IsNaN(multiplier) && !math.IsInf(multiplier, 0)
}

// validateFixedPricingTree enforces one pricing leaf per execution. Without
// this invariant, adding two tiers or multiplying a fixed price by tokens
// would make both the request charge and its billing-unit trace ambiguous.
func validateFixedPricingTree(node ast.Node) error {
	switch part := node.(type) {
	case *ast.ConditionalNode:
		if containsPricingMarker(part.Cond) {
			break
		}
		if err := validateFixedPricingTree(part.Exp1); err != nil {
			return err
		}
		return validateFixedPricingTree(part.Exp2)
	case *ast.BinaryNode:
		if part.Operator != "*" {
			break
		}
		if isRequestPriceMultiplier(part.Right) {
			return validateFixedPricingTree(part.Left)
		}
		if isRequestPriceMultiplier(part.Left) {
			return validateFixedPricingTree(part.Right)
		}
	case *ast.CallNode:
		callee, ok := part.Callee.(*ast.IdentifierNode)
		if !ok || callee.Value != "tier" || len(part.Arguments) != 2 || containsPricingMarker(part.Arguments[0]) {
			break
		}
		price := part.Arguments[1]
		fixed, ok := price.(*ast.CallNode)
		if ok {
			function, direct := fixed.Callee.(*ast.IdentifierNode)
			if direct && function.Value == "fixed" && len(fixed.Arguments) == 1 {
				amount, literal := requestRuleNumber(fixed.Arguments[0])
				if literal && amount >= 0 && !math.IsNaN(amount) && !math.IsInf(amount*1_000_000, 0) {
					return nil
				}
				return fmt.Errorf("fixed price must be a finite, non-negative numeric literal with a finite v1 value")
			}
		}
		if !containsPricingMarker(price) {
			return nil
		}
	}
	return fmt.Errorf("fixed pricing requires tier(name, fixed(amount)) leaves, conditional tiers and request multipliers; token and fixed charges cannot be combined in one leaf")
}
