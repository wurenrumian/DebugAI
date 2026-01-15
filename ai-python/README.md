<!--未来更改python代码后，请在这里描述项目功能和接口-->



## Todos

**注意**：你可以全部使用vibecoding，但是每一阶段都一定要add commit 一次，要写对应的测试函数（phase3不需要，这只是用来展示你的想法）
### phase1 实现调用大模型分析的功能函数

目前先使用deepseek的api吧，比较简单，不用装特别大的包。
接受代码输入和题目输入，和prompt在一起调用api让ai分析
注意调用时让ai进行格式化输出，使用json，这样可以轻松同时获取**评价**，**问题**，**得分**等等你想要的东西
建议把你想要的各种功能拆分实现，不要塞到一个函数里
parse返回json时记得检查格式
返回你定义的返回类型
防范prompt注入
在tests\编写测试代码
在readme中描述功能

### phase2 完善python服务器，使外部请求可以通过fastapi调用功能函数，并获取返回结果

在通过http向python服务器请求时，要包含学生id和对话id
给不同的功能函数分配不同的路由
使用异步调用phase1的功能函数
实现异常处理
在tests\编写测试代码
在readme中描述功能

### phase3 在ai-python文件夹中新建一个文件夹，制作一个简单地html-js-css的本地页面，vibe-coding你的交互想法
