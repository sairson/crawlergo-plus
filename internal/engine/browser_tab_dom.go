package engine

import (
	"context"
	"fmt"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	enums2 "github.com/sairson/crawlergo/internal/engine/enums"
	"github.com/sairson/crawlergo/pkg/utils"
	"os"
	"strings"
	"time"
)

// 这里面处理爬虫的dom操作

type FillForm struct {
	Tab *Tab
}

// AfterDOMLoadedToRunClickAndJavaScript 在dom节点加载完毕后去执行点击和javascript脚本
func (tab *Tab) AfterDOMLoadedToRunClickAndJavaScript() {
	defer tab.WaitGroup.Done()

	// 获取body节点的NodeI,用于之后的节点查找
	if !tab.GetBodyNodeId() {
		return
	}
	// 我们添加dom的等待
	// 这里我们也需要尝试点击所有的a标签,这样才能获取更多的结果

	tab.DomWaitGroup.Add(2)
	go tab.AutoFillFormComponent() // 自动填写表单组件
	go tab.SetObserverJS()         // 添加监听守卫js
	tab.DomWaitGroup.Wait()
	tab.WaitGroup.Add(1)
	go tab.AfterLoadedToRunJavaScript() // 我们等待守卫js加载完成和表单填写完成后触发相关事件
}

// AfterLoadedToRunJavaScript 在加载完成后运行javascript脚本触发事件
func (tab *Tab) AfterLoadedToRunJavaScript() {
	defer tab.WaitGroup.Done()
	tab.FormSubmitWaitGroup.Add(3) // 添加表单提交
	tab.LoadedWaitGroup.Add(3)     // 加载javascript脚本组件
	tab.RemoveList.Add(1)          // 加载监听组件

	go tab.TryToSubmitForm()       // 尝试触发全部的表单提交按钮
	tab.FormSubmitWaitGroup.Wait() // 等待触发完成

	// 如果配置中触发方式是异步的
	if tab.config.EventTriggerMode == enums2.EventTriggerAsync {
		// 触发js级别的函数
		go tab.TriggerJavascriptProtocol()
		go tab.TriggerInlineEvents()
		go tab.TriggerDom2Events()
		tab.LoadedWaitGroup.Wait()
	} else {
		// 我们均按照同步方式触发
		tab.TriggerInlineEvents()
		time.Sleep(tab.config.EventTriggerInterval)
		tab.TriggerDom2Events()
		time.Sleep(tab.config.EventTriggerInterval)
		tab.TriggerJavascriptProtocol()
	}
	// 我们需要等待一段事件使得全部的事件触发后让浏览器发出相关请求
	time.Sleep(tab.config.BeforeExitDelay)

	// 我们移除我们的dom监听器
	go tab.RemoveDOMListener()
	tab.RemoveList.Wait()
}

// TryToSubmitForm 我们尝试提交全部的表单
func (tab *Tab) TryToSubmitForm() {
	tab.SetFormToFrame() // 设置表单的target

	// 接下来尝试全部的提交方法
	go tab.ClickSubmitComponent() // 尝试点击submit组件
	// 尝试点击全部标签页按钮,这是2种方法,
	// 1.触发全部的按钮通过节点
	// 2.通过js事件去触发提交
	go tab.ClickButtonComponent()
	// 尝试点击a标签,这个标签是后期增加的方式
	go tab.ClickHyperlink()
}

// GetBodyNodeId 获取body节点的节点id,用于之后子节点的无等待查询
func (tab *Tab) GetBodyNodeId() bool {
	var docNodeIDs []cdp.NodeID
	ctx := tab.GetCDPExecutor()
	// 设置5秒超时,如果5秒节点还没渲染完成,则直接退出
	tCtx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()
	err := chromedp.NodeIDs(`body`, &docNodeIDs, chromedp.ByQuery).Do(tCtx)
	if len(docNodeIDs) == 0 || err != nil {
		return false
	}
	tab.DocBodyNodeId = docNodeIDs[0]
	return true
}

// AutoFillFormComponent 自动填写表单组件
func (tab *Tab) AutoFillFormComponent() {
	defer tab.DomWaitGroup.Done()
	tab.FillFormWaitGroup.Add(3)
	f := FillForm{
		Tab: tab,
	}

	// 填充输入框
	go func() {
		_ = f.FillFormInput()
	}()
	// 填充文本域
	go func() {
		_ = f.FillTextarea()
	}()
	// 填充复杂的选择框
	go func() {
		_ = f.FillMultiSelect()
	}()
	tab.FillFormWaitGroup.Wait()
}

// FillMultiSelect 填充复杂的选择框
func (f *FillForm) FillMultiSelect() error {
	defer f.Tab.FillFormWaitGroup.Done()
	ctx := f.Tab.GetCDPExecutor()
	tCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	optionNodes, optionErr := f.Tab.GetNodeIDs(`select option:first-child`)
	if optionErr != nil || len(optionNodes) == 0 {
		if optionErr != nil {
			return optionErr
		}
		return nil
	}
	_ = chromedp.SetAttributeValue(optionNodes, "selected", "true", chromedp.ByNodeID).Do(tCtx)
	_ = chromedp.SetJavascriptAttribute(optionNodes, "selected", "true", chromedp.ByNodeID).Do(tCtx)
	return nil
}

// FillTextarea 填充文本域
func (f *FillForm) FillTextarea() error {
	defer f.Tab.FillFormWaitGroup.Done()
	ctx := f.Tab.GetCDPExecutor()
	tCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	value := f.GetMatchInputText("other")
	// 查找文本域标签
	textareaNodes, textareaErr := f.Tab.GetNodeIDs(`textarea`)
	if textareaErr != nil || len(textareaNodes) == 0 {
		if textareaErr != nil {
			return textareaErr
		}
		return nil
	}
	_ = chromedp.SendKeys(textareaNodes, value, chromedp.ByNodeID).Do(tCtx)
	return nil
}

// FillFormInput 填写表单的input组件
func (f *FillForm) FillFormInput() error {
	defer f.Tab.FillFormWaitGroup.Done()
	var nodes []*cdp.Node
	ctx := f.Tab.GetCDPExecutor()
	tCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	// 首先搜索input标签是否存在,如果不存在或存在错误直接退出
	inputNodes, inputErr := f.Tab.GetNodeIDs(`input`)
	if inputErr != nil || len(inputNodes) == 0 {
		if inputErr != nil {
			return inputErr
		}
		return nil
	}
	// 获取所有的input标签
	err := chromedp.Nodes(`input`, &nodes, chromedp.ByQueryAll).Do(tCtx)
	if err != nil {
		return err
	}
	// 遍历全部获取到的节点
	for _, node := range nodes {
		// 我们通过switch-case来确定我们的标签类型
		tCtxN, cancelN := context.WithTimeout(ctx, time.Second*5)
		attrType := node.AttributeValue("type")
		switch {
		case strings.Contains(attrType, "text"): // 这里判断如果input标签属性是text
			inputName := node.AttributeValue("id") + node.AttributeValue("class") + node.AttributeValue("name")
			value := f.GetMatchInputText(inputName)
			var nodeIds = []cdp.NodeID{node.NodeID}
			// 先使用模拟输入
			_ = chromedp.SendKeys(nodeIds, value, chromedp.ByNodeID).Do(tCtxN)
			// 再直接赋值JS属性
			_ = chromedp.SetAttributeValue(nodeIds, "value", value, chromedp.ByNodeID).Do(tCtxN)
		case strings.Contains(attrType, "email") || strings.Contains(attrType, "password") || strings.Contains(attrType, "tel"):
			value := f.GetMatchInputText(attrType)
			var nodeIds = []cdp.NodeID{node.NodeID}
			// 先使用模拟输入
			_ = chromedp.SendKeys(nodeIds, value, chromedp.ByNodeID).Do(tCtxN)
			// 再直接赋值JS属性
			_ = chromedp.SetAttributeValue(nodeIds, "value", value, chromedp.ByNodeID).Do(tCtxN)
		case strings.Contains(attrType, "radio") || strings.Contains(attrType, "checkbox"):
			var nodeIds = []cdp.NodeID{node.NodeID}
			// 为每一个检查框都设置为true属性
			_ = chromedp.SetAttributeValue(nodeIds, "checked", "true", chromedp.ByNodeID).Do(tCtxN)
		case strings.Contains(attrType, "file") || strings.Contains(attrType, "image"):
			var nodeIds = []cdp.NodeID{node.NodeID}
			wd, _ := os.Getwd()
			filePath := wd + "/upload/image.png"
			_ = chromedp.RemoveAttribute(nodeIds, "accept", chromedp.ByNodeID).Do(tCtxN)
			_ = chromedp.RemoveAttribute(nodeIds, "required", chromedp.ByNodeID).Do(tCtxN)
			_ = chromedp.SendKeys(nodeIds, filePath, chromedp.ByNodeID).Do(tCtxN)
		default:
			// 其他的我们都按照text文本解析
			inputName := node.AttributeValue("id") + node.AttributeValue("class") + node.AttributeValue("name")
			value := f.GetMatchInputText(inputName)
			var nodeIds = []cdp.NodeID{node.NodeID}
			// 先使用模拟输入
			_ = chromedp.SendKeys(nodeIds, value, chromedp.ByNodeID).Do(tCtxN)
			// 再直接赋值JS属性
			_ = chromedp.SetAttributeValue(nodeIds, "value", value, chromedp.ByNodeID).Do(tCtxN)
		}
		cancelN()
	}
	return nil
}

// GetMatchInputText 获取输入的表单值
func (f *FillForm) GetMatchInputText(name string) string {
	// 如果自定义了关键词，模糊匹配
	for key, value := range f.Tab.config.CustomFormKeywordValues {
		if strings.Contains(name, key) {
			return value
		}
	}

	name = strings.ToLower(name)
	// 我们通过默认值来输入
	for key, item := range enums2.DefaultInputTextMap {
		for _, keyword := range item["keyword"].([]string) {
			if strings.Contains(name, keyword) {
				if customValue, ok := f.Tab.config.CustomFormValues[key]; ok {
					return customValue
				} else {
					return item["value"].(string)
				}
			}
		}
	}
	return f.Tab.config.CustomFormValues["default"]
}

// GetNodeIDs 立即根据条件获取Nodes的ID，不等待
func (tab *Tab) GetNodeIDs(sel string) ([]cdp.NodeID, error) {
	ctx := tab.GetCDPExecutor()
	return dom.QuerySelectorAll(tab.DocBodyNodeId, sel).Do(ctx)
}

// EvaluateWithNode 用节点来执行表达式
func (tab *Tab) EvaluateWithNode(expression string, node *cdp.Node) error {
	ctx := tab.GetCDPExecutor()
	var res bool
	err := chromedp.EvaluateAsDevTools(enums2.Snippet(expression, enums2.CashX(true), "", node), &res).Do(ctx)
	if err != nil {
		return err
	}
	return nil
}

// DismissDialog 对话框
func (tab *Tab) DismissDialog() {
	defer tab.WaitGroup.Done()
	ctx := tab.GetCDPExecutor()
	_ = page.HandleJavaScriptDialog(false).Do(ctx)
}

// Evaluate  执行js表达式
func (tab *Tab) Evaluate(expression string) {
	ctx := tab.GetCDPExecutor()
	tCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	_, exception, err := runtime.Evaluate(expression).Do(tCtx)
	if exception != nil {
		return
	}
	if err != nil {
		return
	}
}

// SetObserverJS 设置观察js函数
func (tab *Tab) SetObserverJS() {
	defer tab.DomWaitGroup.Done()
	// 设置Dom节点变化的观察函数
	go tab.Evaluate(enums2.ObserverJS)
}

// SetFormToFrame 设置表单为指定的模板
func (tab *Tab) SetFormToFrame() {
	// 新建一个frame表单模板
	var name = utils.RandSeq(8)
	tab.Evaluate(fmt.Sprintf(enums2.NewFrameTemplate, name, name))
	// 接下来将所有的 form 节点target都指向它
	ctx := tab.GetCDPExecutor()
	formNodes, formErr := tab.GetNodeIDs(`form`)
	if formErr != nil || len(formNodes) == 0 {
		if formErr != nil {
			return
		}
		return
	}
	tCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	_ = chromedp.SetAttributeValue(formNodes, "target", name, chromedp.ByNodeID).Do(tCtx)
}

// ClickSubmitComponent  点击按钮 type = submit
func (tab *Tab) ClickSubmitComponent() {
	defer tab.FormSubmitWaitGroup.Done()

	// 首先点击按钮 type=submit
	ctx := tab.GetCDPExecutor()

	// 获取所有的form节点 直接执行submit
	formNodes, formErr := tab.GetNodeIDs(`form`)
	if formErr != nil || len(formNodes) == 0 {
		if formErr != nil {
			return
		}
		return
	}
	tCtx1, cancel1 := context.WithTimeout(ctx, time.Second*2)
	defer cancel1()
	_ = chromedp.Submit(formNodes, chromedp.ByNodeID).Do(tCtx1)

	// 获取所有的input标签
	inputNodes, inputErr := tab.GetNodeIDs(`form input[type=submit]`)
	if inputErr != nil || len(inputNodes) == 0 {
		if inputErr != nil {
			return
		}
		return
	}
	tCtx2, cancel2 := context.WithTimeout(ctx, time.Second*2)
	defer cancel2()
	_ = chromedp.Click(inputNodes, chromedp.ByNodeID).Do(tCtx2)
}

// ClickHyperlink 尝试点击全部的a标签,但是这个a标签不能随便点击,否则会超出当前的爬取范围到其他页面当中
func (tab *Tab) ClickHyperlink() {
	defer tab.FormSubmitWaitGroup.Done()
	ctx := tab.GetCDPExecutor()
	var href []map[string]string
	tCtx, cancel := context.WithTimeout(ctx, time.Second*1)
	_ = chromedp.AttributesAll(`a[href]`, &href, chromedp.ByQueryAll).Do(tCtx)
	cancel()
	// 我们获取到全部的a href 标签
	for _, v := range href {
		// 如果我们获取的a标签不包含https或者http的话,证明是当前网站的链接
		if !strings.Contains(v["href"], "https://") && !strings.Contains(v["href"], "http://") && !tab.HrefClick.Contains(v["href"]) {
			// 点击这个属性的按钮
			tab.HrefClick.Add(v["href"])
		} else {
			// 包含https或者http，并且根域也是当前的根域
			if strings.Contains(v["href"], tab.config.RootDomain) && !tab.HrefClick.Contains(v["href"]) {
				tab.HrefClick.Add(v["href"])
			}
		}
	}
	// 遍历我们点击的按钮指定链接的按钮
	for _, h := range tab.HrefClick.ToSlice() {
		hrefNodes, err := tab.GetNodeIDs(fmt.Sprintf(`a[href="%s"]`, h))
		if err != nil {
			continue
		}
		tCtx2, cancel2 := context.WithTimeout(ctx, time.Second*2)
		_ = chromedp.Click(hrefNodes, chromedp.ByNodeID).Do(tCtx2) // 尝试点击这些a标签
		cancel2()
	}
}

// ClickButtonComponent 点击来自tab页的全部按钮
func (tab *Tab) ClickButtonComponent() {
	defer tab.FormSubmitWaitGroup.Done()
	// 首先获取当前tab的上下文
	ctx := tab.GetCDPExecutor()
	// 查找全部的的表单按钮
	btnNodeIDs, bErr := tab.GetNodeIDs(`form button`)
	// 如果存在错误或者按钮数量为0的时候
	if bErr != nil || len(btnNodeIDs) == 0 {
		return
	}
	tCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	// 点击全部的按钮

	_ = chromedp.Click(btnNodeIDs, chromedp.ByNodeID).Do(tCtx)

	// 使用JS的click方法进行点击
	var btnNodes []*cdp.Node
	tCtx2, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	err := chromedp.Nodes(btnNodeIDs, &btnNodes, chromedp.ByNodeID).Do(tCtx2)
	if err != nil {
		return
	}
	for _, node := range btnNodes {
		_ = tab.EvaluateWithNode(enums2.FormNodeClickJS, node)
	}
}

// TriggerJavascriptProtocol 触发javascript的伪协议
func (tab *Tab) TriggerJavascriptProtocol() {
	defer tab.LoadedWaitGroup.Done()
	tab.Evaluate(fmt.Sprintf(enums2.TriggerJavascriptProtocol,
		tab.config.EventTriggerInterval.Seconds()*1000,
		tab.config.EventTriggerInterval.Seconds()*1000),
	)
}

// TriggerInlineEvents 触发所有的内敛事件
func (tab *Tab) TriggerInlineEvents() {
	defer tab.LoadedWaitGroup.Done()
	tab.Evaluate(fmt.Sprintf(enums2.TriggerInlineEventJS, tab.config.EventTriggerInterval.Seconds()*1000))
}

// TriggerDom2Events 触发dom 2级事件
func (tab *Tab) TriggerDom2Events() {
	defer tab.LoadedWaitGroup.Done()
	tab.Evaluate(fmt.Sprintf(enums2.TriggerDom2EventJS, tab.config.EventTriggerInterval.Seconds()*1000))
}

// RemoveDOMListener 移除dom节点变化监听
func (tab *Tab) RemoveDOMListener() {
	defer tab.RemoveList.Done()
	// 移除DOM节点变化监听
	tab.Evaluate(enums2.RemoveDOMListenerJS)
}
