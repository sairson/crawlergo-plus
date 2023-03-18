# crawlergo-plus
爬虫的暴力美学，在projectdiscover和原版crawlergo的基础上修改而来，目前只提供了测试，还没有完全修改好
<br> 1. 增加了对页面`<a href="#">`标签的暴力点击，但是做了对根域的判断，不会点击的超范围
<br> 2. 增加了sitemap.xml的解析功能
<br> 3. 增加了对响应头的解析
<br> 4. 增加了对更多链接属性，比如src,href,link,background等等
<br> 5. 增加了结果回调，但是如果用这个的话，不会触发过滤函数，结果会更复杂
<br> 6. 增加自定义正则，这点还在完善


# 注意和测试
1. 经过对projectdiscover的katana的测试(参数仅使用 katana -u https://security-crawl-maze.app  -json) 和原版crawlergo (smart智能过滤不启用robots.txt解析和路径fuzz，填充post为username=admin&password=password) 对 https://security-crawl-maze.app 爬取以及crawlergo-plus(启用robots.txt,sitemap.xml,链接全点击,post参数为username=admin&password=password，以及采用noheadless，simple过滤模式)

<br>katana: 15条结果(不重复)
<br>crawlergo: 12条结果(不重复)
<br>crawlergo-plus: 711条(不重复,存在不可用链接，但是页面.found基本全部找到)


![1679104322028](https://user-images.githubusercontent.com/74412075/226077061-505115df-30e9-4ee1-8595-43a00efb1ee4.png)
