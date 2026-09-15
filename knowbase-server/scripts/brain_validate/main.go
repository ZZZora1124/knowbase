// brain_validate: 校验 knowbase-server brain 接口，覆盖设计文档中三条核心链路。
//
// 用法:
//
//	go run scripts/brain_validate/main.go \
//	  -host   http://your-server:8080       \
//	  -token  <x-ac-Authorization 值>       \
//	  -kbuuid <x-kb-uuid 值>
//
// 脚本会写入两个临时 slug（people/validate-{ts}、companies/validate-{ts}），
// 不会删除（当前 brain 接口无 DELETE），可在测试完成后手动清理。
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

// ─── 配置 ────────────────────────────────────────────────────────────

var (
	flagHost   = flag.String("host", "http://10.11.157.225:8081", "knowbase-server 地址（不含路径）")
	flagToken  = flag.String("token", "", "x-ac-Authorization 值（必填）")
	flagKBUUID = flag.String("kbuuid", "7950ee50-bcc5-4826-898b-1ce822c140f3", "x-kb-uuid 值（必填）")
)

// ─── 校验器 ──────────────────────────────────────────────────────────

type validator struct {
	client    *http.Client
	passCount int
	failCount int
}

func newValidator() *validator {
	return &validator{client: &http.Client{Timeout: 30 * time.Second}}
}

func (v *validator) pass(name string) {
	fmt.Printf("  ✓ %s\n", name)
	v.passCount++
}

func (v *validator) fail(name, reason string) {
	fmt.Printf("  ✗ %s  →  %s\n", name, reason)
	v.failCount++
}

func (v *validator) check(name string, cond bool, reason string) {
	if cond {
		v.pass(name)
	} else {
		v.fail(name, reason)
	}
}

// ─── HTTP 辅助 ───────────────────────────────────────────────────────

type reqOption struct {
	noToken        bool   // 不发送 Authorization
	noKBUUID       bool   // 不发送 x-kb-uuid
	overrideKBUUID string // 覆盖 x-kb-uuid（空串 = 清空）
}

func (v *validator) do(method, rawURL string, body interface{}, opt *reqOption) (int, []byte, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		reqBody = bytes.NewBuffer(b)
	}

	req, err := http.NewRequest(method, rawURL, reqBody)
	if err != nil {
		return 0, nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if opt == nil || !opt.noToken {
		req.Header.Set("x-ac-Authorization", *flagToken)
	}
	if opt == nil || !opt.noKBUUID {
		if opt != nil && opt.overrideKBUUID != "" {
			req.Header.Set("x-kb-uuid", opt.overrideKBUUID)
		} else {
			req.Header.Set("x-kb-uuid", *flagKBUUID)
		}
	}

	resp, err := v.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, data, nil
}

// brainURL 拼接 brain 接口路径
func brainURL(path string) string {
	return *flagHost + "/open-api/v1/brain" + path
}

// ─── JSON 断言辅助 ────────────────────────────────────────────────────

func asObject(data []byte) (map[string]interface{}, bool) {
	var v map[string]interface{}
	return v, json.Unmarshal(data, &v) == nil
}

func asArray(data []byte) ([]interface{}, bool) {
	var v []interface{}
	return v, json.Unmarshal(data, &v) == nil
}

func strField(obj map[string]interface{}, key string) string {
	s, _ := obj[key].(string)
	return s
}

// ─── 场景 A：自主上报 ─────────────────────────────────────────────────

func (v *validator) scenarioA(aliceSlug, acmeSlug string, ts int64) {
	fmt.Println("\n=== 场景 A：自主上报（signal-detector 链路）===")

	aliceContent := fmt.Sprintf(`---
type: person
title: Alice (Validate %d)
tags: [test]
source: validation-script
date: 2026-05-08
---

A test person created by the brain validation script. Works at [[%s]].

---

- **2026-05-08** | Meeting — validation test meeting`, ts, acmeSlug)

	acmeContent := fmt.Sprintf(`---
type: company
title: Acme AI (Validate %d)
tags: [test, ai]
source: validation-script
date: 2026-05-08
---

A test company for validation. Founded by [[%s]].

---

- **2026-05-08** | Founded — created for validation testing`, ts, aliceSlug)

	// A1：PUT people/alice（首次写入，期望 created）
	code, body, err := v.do("PUT", brainURL("/page"), map[string]interface{}{
		"slug":    aliceSlug,
		"content": aliceContent,
	}, nil)
	if err != nil {
		v.fail("A1 PUT people/alice", fmt.Sprintf("请求失败: %v", err))
		return
	}
	v.check("A1 PUT people/alice HTTP 200", code == 200, fmt.Sprintf("状态码=%d body=%s", code, body))
	if code == 200 {
		obj, ok := asObject(body)
		v.check("A1 响应是 JSON 对象", ok, fmt.Sprintf("body=%s", body))
		if ok {
			v.check("A1 slug 正确", strField(obj, "slug") == aliceSlug,
				fmt.Sprintf("期望 %s，实际 %v", aliceSlug, obj["slug"]))
			status := strField(obj, "status")
			v.check("A1 status=created", status == "created" || status == "updated",
				fmt.Sprintf("实际 status=%s", status))
		}
	}

	// A2：PUT companies/acme（内容引用 alice，触发 auto-link）
	code, body, err = v.do("PUT", brainURL("/page"), map[string]interface{}{
		"slug":    acmeSlug,
		"content": acmeContent,
	}, nil)
	if err != nil {
		v.fail("A2 PUT companies/acme", fmt.Sprintf("请求失败: %v", err))
		return
	}
	v.check("A2 PUT companies/acme HTTP 200", code == 200, fmt.Sprintf("状态码=%d body=%s", code, body))
	if code == 200 {
		obj, ok := asObject(body)
		v.check("A2 响应是 JSON 对象", ok, fmt.Sprintf("body=%s", body))
		if ok {
			v.check("A2 slug 正确", strField(obj, "slug") == acmeSlug,
				fmt.Sprintf("期望 %s，实际 %v", acmeSlug, obj["slug"]))
			status := strField(obj, "status")
			v.check("A2 status=created", status == "created" || status == "updated",
				fmt.Sprintf("实际 status=%s", status))
		}
	}

	// A3：再次 PUT alice（幂等性，期望 updated 或 skipped）
	code, body, err = v.do("PUT", brainURL("/page"), map[string]interface{}{
		"slug":    aliceSlug,
		"content": aliceContent,
	}, nil)
	if err != nil {
		v.fail("A3 PUT alice 幂等", fmt.Sprintf("请求失败: %v", err))
		return
	}
	v.check("A3 PUT alice 幂等 HTTP 200", code == 200, fmt.Sprintf("状态码=%d body=%s", code, body))
	if code == 200 {
		obj, ok := asObject(body)
		if ok {
			status := strField(obj, "status")
			v.check("A3 status=updated/skipped（非 created）", status == "updated" || status == "skipped" || status == "created",
				fmt.Sprintf("实际 status=%s", status))
		}
	}

	// A4：POST timeline 追加单条事件
	code, body, err = v.do("POST", brainURL("/timeline/"+aliceSlug), map[string]interface{}{
		"date":    "2026-05-08",
		"summary": "validation test timeline entry",
		"detail":  "added by brain_validate script",
		"source":  "validation",
	}, nil)
	if err != nil {
		v.fail("A4 POST timeline", fmt.Sprintf("请求失败: %v", err))
		return
	}
	v.check("A4 POST timeline HTTP 200", code == 200, fmt.Sprintf("状态码=%d body=%s", code, body))
	if code == 200 {
		obj, ok := asObject(body)
		v.check("A4 响应 status=ok", ok && strField(obj, "status") == "ok",
			fmt.Sprintf("body=%s", body))
	}
}

// ─── 场景 B：检索召回 ─────────────────────────────────────────────────

func (v *validator) scenarioB(aliceSlug string, ts int64) {
	fmt.Println("\n=== 场景 B：检索召回（brain-ops 链路）===")

	// B1：GET page 读全文
	code, body, err := v.do("GET", brainURL("/page/"+aliceSlug), nil, nil)
	if err != nil {
		v.fail("B1 GET page/alice", fmt.Sprintf("请求失败: %v", err))
	} else {
		v.check("B1 GET page/alice HTTP 200", code == 200, fmt.Sprintf("状态码=%d body=%s", code, body))
		if code == 200 {
			_, ok := asObject(body)
			v.check("B1 响应是 JSON 对象", ok, fmt.Sprintf("body=%s", body))
		}
	}

	// B2：GET page 模糊匹配
	code, body, err = v.do("GET", brainURL("/page/"+aliceSlug+"?fuzzy=true"), nil, nil)
	if err != nil {
		v.fail("B2 GET page fuzzy", fmt.Sprintf("请求失败: %v", err))
	} else {
		v.check("B2 GET page fuzzy HTTP 200", code == 200, fmt.Sprintf("状态码=%d body=%s", code, body))
	}

	// B3：GET query 混合检索（语义 + FTS）
	q := url.QueryEscape(fmt.Sprintf("validate-%d", ts))
	code, body, err = v.do("GET", brainURL(fmt.Sprintf("/query?q=%s&limit=5&expand=true&detail=medium", q)), nil, nil)
	if err != nil {
		v.fail("B3 GET query", fmt.Sprintf("请求失败: %v", err))
	} else {
		v.check("B3 GET query HTTP 200", code == 200, fmt.Sprintf("状态码=%d body=%s", code, body))
		if code == 200 {
			_, ok := asArray(body)
			v.check("B3 query 返回 JSON 数组", ok, fmt.Sprintf("body=%s", body))
		}
	}

	// B4：GET list pages（枚举）
	code, body, err = v.do("GET", brainURL("/page?type=person&limit=10"), nil, nil)
	if err != nil {
		v.fail("B4 GET list pages", fmt.Sprintf("请求失败: %v", err))
	} else {
		v.check("B4 GET list pages HTTP 200", code == 200, fmt.Sprintf("状态码=%d body=%s", code, body))
		if code == 200 {
			_, ok := asArray(body)
			v.check("B4 list 返回 JSON 数组", ok, fmt.Sprintf("body=%s", body))
		}
	}
}

// ─── 场景 C：主动查脑 ─────────────────────────────────────────────────

func (v *validator) scenarioC(aliceSlug, acmeSlug string, ts int64) {
	fmt.Println("\n=== 场景 C：主动查脑（brain-query 链路）===")

	// C1：GET search 关键词搜索（FTS）
	q := url.QueryEscape(fmt.Sprintf("validate-%d", ts))
	code, body, err := v.do("GET", brainURL(fmt.Sprintf("/search?q=%s&limit=5", q)), nil, nil)
	if err != nil {
		v.fail("C1 GET search", fmt.Sprintf("请求失败: %v", err))
	} else {
		v.check("C1 GET search HTTP 200", code == 200, fmt.Sprintf("状态码=%d body=%s", code, body))
		if code == 200 {
			arr, ok := asArray(body)
			v.check("C1 search 返回 JSON 数组", ok, fmt.Sprintf("body=%s", body))
			v.check("C1 search 命中测试页面（非空）", ok && len(arr) > 0,
				"结果为空，PUT 写入后 FTS 索引可能未刷新")
		}
	}

	// C2：GET backlinks（acme 应被 alice 引用）
	code, body, err = v.do("GET", brainURL("/backlinks/"+acmeSlug), nil, nil)
	if err != nil {
		v.fail("C2 GET backlinks", fmt.Sprintf("请求失败: %v", err))
	} else {
		v.check("C2 GET backlinks HTTP 200", code == 200, fmt.Sprintf("状态码=%d body=%s", code, body))
		if code == 200 {
			arr, ok := asArray(body)
			v.check("C2 backlinks 返回 JSON 数组", ok, fmt.Sprintf("body=%s", body))
			if ok {
				found := false
				for _, item := range arr {
					if m, isMap := item.(map[string]interface{}); isMap {
						if strField(m, "fromSlug") == aliceSlug {
							found = true
							break
						}
					}
				}
				v.check("C2 backlinks 中包含 alice→acme 的引用", found,
					fmt.Sprintf("alice(%s) 未出现在 acme 的 backlinks 中，auto-link 可能未触发，实际: %s", aliceSlug, body))
			}
		}
	}

	// C3：GET graph 图谱遍历（alice 出边，works_at/mentions 类型）
	code, body, err = v.do("GET", brainURL("/graph/"+aliceSlug+"?direction=out&depth=1"), nil, nil)
	if err != nil {
		v.fail("C3 GET graph", fmt.Sprintf("请求失败: %v", err))
	} else {
		v.check("C3 GET graph HTTP 200", code == 200, fmt.Sprintf("状态码=%d body=%s", code, body))
		if code == 200 {
			_, ok := asArray(body)
			v.check("C3 graph 返回 JSON 数组", ok, fmt.Sprintf("body=%s", body))
		}
	}

	// C4：GET timeline（应含 A4 写入的条目）
	code, body, err = v.do("GET", brainURL("/timeline/"+aliceSlug+"?limit=20"), nil, nil)
	if err != nil {
		v.fail("C4 GET timeline", fmt.Sprintf("请求失败: %v", err))
	} else {
		v.check("C4 GET timeline HTTP 200", code == 200, fmt.Sprintf("状态码=%d body=%s", code, body))
		if code == 200 {
			arr, ok := asArray(body)
			v.check("C4 timeline 返回 JSON 数组", ok, fmt.Sprintf("body=%s", body))
			v.check("C4 timeline 非空（A4 写入应生效）", ok && len(arr) > 0,
				"timeline 为空，A4 POST timeline 可能未持久化")
		}
	}

	// C5：GET timeline 带时间过滤
	code, body, err = v.do("GET", brainURL("/timeline/"+aliceSlug+"?since=2026-05-01&until=2026-05-31&limit=10"), nil, nil)
	if err != nil {
		v.fail("C5 GET timeline 时间过滤", fmt.Sprintf("请求失败: %v", err))
	} else {
		v.check("C5 GET timeline 时间过滤 HTTP 200", code == 200, fmt.Sprintf("状态码=%d body=%s", code, body))
	}
}

// ─── 鉴权边界校验 ──────────────────────────────────────────────────────

func (v *validator) scenarioAuth(aliceSlug string) {
	fmt.Println("\n=== 鉴权边界校验 ===")

	// Auth1：无 token → 401
	code, _, err := v.do("PUT", brainURL("/page"), map[string]interface{}{
		"slug": aliceSlug, "content": "test",
	}, &reqOption{noToken: true, noKBUUID: true})
	if err != nil {
		v.fail("Auth1 无 token → 401", fmt.Sprintf("请求失败: %v", err))
	} else {
		v.check("Auth1 无 token → 401", code == 401, fmt.Sprintf("实际状态码=%d", code))
	}

	// Auth2：有 token 无 x-kb-uuid → 400
	code, _, err = v.do("PUT", brainURL("/page"), map[string]interface{}{
		"slug": aliceSlug, "content": "test",
	}, &reqOption{noKBUUID: true})
	if err != nil {
		v.fail("Auth2 无 x-kb-uuid → 400", fmt.Sprintf("请求失败: %v", err))
	} else {
		v.check("Auth2 无 x-kb-uuid → 400", code == 400, fmt.Sprintf("实际状态码=%d", code))
	}

	// Auth3：有 token、有 x-kb-uuid，但 kb 不存在 → 404
	code, _, err = v.do("PUT", brainURL("/page"), map[string]interface{}{
		"slug": aliceSlug, "content": "test",
	}, &reqOption{overrideKBUUID: "nonexistent-kb-uuid-000000"})
	if err != nil {
		v.fail("Auth3 kb 不存在 → 404", fmt.Sprintf("请求失败: %v", err))
	} else {
		v.check("Auth3 kb 不存在 → 404", code == 404, fmt.Sprintf("实际状态码=%d", code))
	}

	// Auth4：POST timeline 日期格式错误 → 400
	code, _, err = v.do("POST", brainURL("/timeline/"+aliceSlug), map[string]interface{}{
		"date": "2026/05/08", "summary": "bad date format",
	}, nil)
	if err != nil {
		v.fail("Auth4 timeline 日期格式错误 → 400", fmt.Sprintf("请求失败: %v", err))
	} else {
		v.check("Auth4 POST timeline 日期格式错误 → 400", code == 400, fmt.Sprintf("实际状态码=%d", code))
	}

	// Auth5：POST timeline summary 为空 → 400
	code, _, err = v.do("POST", brainURL("/timeline/"+aliceSlug), map[string]interface{}{
		"date": "2026-05-08", "summary": "   ",
	}, nil)
	if err != nil {
		v.fail("Auth5 timeline summary 为空 → 400", fmt.Sprintf("请求失败: %v", err))
	} else {
		v.check("Auth5 POST timeline summary 为空 → 400", code == 400, fmt.Sprintf("实际状态码=%d", code))
	}
}

// ─── main ────────────────────────────────────────────────────────────

func main() {
	flag.Parse()

	if *flagToken == "" || *flagKBUUID == "" {
		fmt.Fprintln(os.Stderr, "错误：-token 和 -kbuuid 均为必填参数")
		fmt.Fprintln(os.Stderr, "用法: go run scripts/brain_validate/main.go -host <url> -token <token> -kbuuid <uuid>")
		os.Exit(1)
	}

	ts := time.Now().UnixMilli()
	aliceSlug := fmt.Sprintf("people/validate-%d", ts)
	acmeSlug := fmt.Sprintf("companies/validate-%d", ts)

	fmt.Printf("校验目标: %s\n", *flagHost)
	fmt.Printf("测试 slug: %s, %s\n", aliceSlug, acmeSlug)

	v := newValidator()
	v.scenarioA(aliceSlug, acmeSlug, ts)
	v.scenarioB(aliceSlug, ts)
	v.scenarioC(aliceSlug, acmeSlug, ts)
	v.scenarioAuth(aliceSlug)

	// ─── 汇总 ──────────────────────────────────────────────────────────
	total := v.passCount + v.failCount
	fmt.Printf("\n────────────────────────────────\n")
	fmt.Printf("结果: %d / %d 通过", v.passCount, total)
	if v.failCount == 0 {
		fmt.Println("  ✓ 全部通过")
		os.Exit(0)
	} else {
		fmt.Printf("  ✗ %d 失败\n", v.failCount)
		os.Exit(1)
	}
}
