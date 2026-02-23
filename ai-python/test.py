#可直接执行，请确保设置了DEEPSEEK_API_KEY环境变量，同目录内有存放测试样例文件夹test_cases。
import os
import json
import asyncio
import aiohttp
from pathlib import Path
from typing import Dict, List, Optional
import time
from dataclasses import dataclass
from datetime import datetime

@dataclass
class TestCase:
    """测试用例数据结构"""
    case_id: str
    problem_description: str
    code: str
    test_points: List[Dict]
    
@dataclass
class TestPoint:
    """测试点数据结构"""
    input: str
    expected_output: str
    status: str = "Pending"
    actual_output: str = ""
    error_message: str = ""

class DirectAITester:
    def __init__(self, max_concurrent: int = 10):
        """
        初始化直接AI测试器
        
        Args:
            api_key: DeepSeek API密钥
            max_concurrent: 最大并发数
        """
        self.api_url = "https://api.deepseek.com/v1/chat/completions"#"https://api.siliconflow.cn/v1/chat/completions"
        self.api_key = self._get_api_key_from_env()
        self.max_concurrent = max_concurrent
        self.results = []
        self.semaphore = asyncio.Semaphore(max_concurrent)
        
    def _get_api_key_from_env(self) -> Optional[str]:
        """从环境变量获取API密钥"""
        return os.environ.get("DEEPSEEK_API_KEY")
    
    def set_api_key(self, api_key: str):
        """设置API密钥"""
        self.api_key = api_key
        
    def read_case_folder(self, case_path: Path) -> Optional[TestCase]:
        """
        读取单个case文件夹的内容
        
        Args:
            case_path: case文件夹路径
            
        Returns:
            TestCase对象，如果读取失败返回None
        """
        try:
            case_id = case_path.name
            
            # 读取题目描述（假设是txt文件）
            desc_files = list(case_path.glob("*.txt"))
            if not desc_files:
                print(f"⚠️ {case_id}: 未找到题目描述文件")
                return None
            with open(desc_files[0], 'r', encoding='utf-8') as f:
                problem_description = f.read()
            
            # 读取学生代码（排除copy.c文件）
            code_files = [f for f in case_path.glob("*.c") if "copy" not in f.name.lower()]
            if not code_files:
                print(f"⚠️ {case_id}: 未找到学生代码文件")
                return None
            with open(code_files[0], 'r', encoding='utf-8') as f:
                code = f.read()
            
            # 读取测试点数据
            json_files = list(case_path.glob("*.json"))
            if not json_files:
                print(f"⚠️ {case_id}: 未找到测试点JSON文件")
                test_points = []
            else:
                with open(json_files[0], 'r', encoding='utf-8') as f:
                    test_points = json.load(f)
            
            return TestCase(
                case_id=case_id,
                problem_description=problem_description,
                code=code,
                test_points=test_points
            )
            
        except Exception as e:
            print(f"❌ {case_id}: 读取失败 - {str(e)}")
            return None
    
    def load_all_cases(self, test_dir: str = "test_cases") -> List[TestCase]:
        """
        加载所有测试用例
        
        Args:
            test_dir: 测试根目录
            
        Returns:
            测试用例列表
        """
        test_path = Path(test_dir)
        if not test_path.exists():
            raise FileNotFoundError(f"测试目录不存在: {test_dir}")
        
        cases = []
        # 遍历所有case文件夹
        for item in sorted(test_path.iterdir()):
            if item.is_dir() and item.name.startswith("case"):
                print(f"📂 读取 {item.name}...")
                case = self.read_case_folder(item)
                if case:
                    cases.append(case)
        
        print(f"\n✅ 成功加载 {len(cases)} 个测试用例")
        return cases
    
    def _format_test_info(self, test_points: List[Dict]) -> str:
        """格式化测试点信息"""
        if not test_points:
            return "无测试点"
        
        test_info = "\n测试点信息："
        failed_count = 0
        
        for i, test_point in enumerate(test_points):
            if i < 20:  # 只显示前20个测试点的详细信息
                input_str = test_point.get('input', '')[:80]
                if len(test_point.get('input', '')) > 80:
                    input_str += "... [已截断]"
                
                status = test_point.get('status', 'Unknown')
                test_info += f"\n测试点 {i+1}: 状态={status}"
                test_info += f"\n  输入: {input_str}"
            
            if test_point.get('status') != "Accepted":
                failed_count += 1
        
        # 汇总信息
        test_info += f"\n\n测试点汇总：共{len(test_points)}个测试点，"
        test_info += f"通过{len(test_points)-failed_count}个，失败{failed_count}个"
        
        return test_info
    
    def create_evaluation_prompt(self, case: TestCase) -> str:
        """创建评价提示词"""
        test_info = self._format_test_info(case.test_points)
        
        prompt = f"""
题目要求：
{case.problem_description}

学生代码（C/C++）：
```
{case.code}
```

测试点通过情况：
{test_info}
"""
        return prompt

    async def submit_single(self, session: aiohttp.ClientSession, 
                        case: TestCase) -> Dict:
        """
        提交单个测试用例到DeepSeek API
        
        Args:
            session: aiohttp会话
            case: 测试用例
            
        Returns:
            响应结果
        """
        async with self.semaphore:  # 使用信号量控制并发
            start_time = time.time()
            
            prompt = self.create_evaluation_prompt(case)
            
            # 构建DeepSeek API请求数据
            request_data = {
                "model": "deepseek-chat",
                "messages": [
                    {
                        "role": "system",
                        "content": f"""
请你作为编程教学助手，结合测试点通过情况对编程初学者的代码进行评价。请严格按照JSON格式返回结果。

请按照以下标准，从4个维度进行评价，请重点关注学生思路和代码的基本功能实现，对代码规范和效率不要过于严格：
1.功能正确（该维度重点分析学生思路与题目要求功能的照应，学生代码实现过程中的错误不影响本项评价，即本维度只评价“做了没”，不评价“做对没”。）
优秀：无语法错误，**学生思路**照应了题目要求的所有基本功能，满足题目对特定函数的使用要求（如有）。
合格：语法错误不超过3种(注意是3种不是3处)，**学生思路**实现了主要功能但存在**核心功能**偏差或遗漏。
待改进：存在>3种语法错误，或严重偏离题意，未能实现题目规定的主要功能。
2.逻辑严谨（学生无需考虑题目说明输入格式之外的异常情况）
优秀：覆盖常见边界条件和异常情况，对数组、递归、函数等的运用无漏洞。
合格：对至少1种边界条件进行处理，逻辑漏洞少于3处，有溢出等导致的错误。
待改进：缺乏边界条件和异常处理，或逻辑有多处明显漏洞。
3.算法效率（该维度重点分析学生算法选择在效率的合理性，因逻辑错误导致的超时不影响本项评价）
优秀：算法效率合理，时间/空间复杂度正常，冗余计算少。
合格：算法效率可接受，时间/空间复杂度在可接受范围，冗余计算较多。
待改进：算法效率低下，时间/空间复杂度过高，因效率所致超时/超内存测试点多。
4.结构规范（若题目中直接给出部分变量名，学生可直接使用，无需考虑规范性）
优秀：命名规范且表意清晰，代码结构层次分明，可读性好。
合格：命名基本规范（以连续字母a,b,c,d命名也可接受），代码结构较清晰，但存在局部混乱。
待改进：命名随意或无意义，代码结构混乱，可读性差。

**重要说明**：
- 四个维度的评价必须严格独立，互不影响
- 一个维度的缺陷（如逻辑错误）不应影响其他维度的评价
- 分析时请明确问题所属的具体维度，不要将其他维度的问题归因到当前维度。例如：逻辑错误导致的测试点失败，不应降低算法效率的等级

注意：分析要精简一点。

请按以下JSON格式返回：
{{
    "overall_evaluation": "<整体评价>",
    "functional_correctness": {{
        "grade": "<优秀/合格/待改进>",
        "analysis": "<具体分析>"
    }},
    "logical_rigor": {{
        "grade": "<优秀/合格/待改进>",
        "analysis": "<具体分析>"
    }},
    "algorithm_quality": {{
        "grade": "<优秀/合格/待改进>",
        "analysis": "<具体分析>"
    }},
    "structural_normativity": {{
        "grade": "<优秀/合格/待改进>",
        "analysis": "<具体分析>"
    }},
}}
"""
                    },
                    {
                        "role": "user",
                        "content": prompt
                    }
                ],
                "temperature": 0.3,
                "max_tokens": 8192,
                #"enable_thinking": True,#新增
                "response_format": {"type": "json_object"}
            }
            
            headers = {
                "Content-Type": "application/json",
                "Authorization": f"Bearer {self.api_key}"
            }
            
            try:
                async with session.post(self.api_url, 
                                    json=request_data, 
                                    headers=headers) as response:
                    elapsed = time.time() - start_time
                    
                    if response.status == 200:
                        result = await response.json()
                        # 提取AI返回的内容
                        ai_response = result['choices'][0]['message']['content']
                        
                        # 解析JSON响应
                        try:
                            evaluation_result = json.loads(ai_response)
                            print(f"✅ {case.case_id}: 成功 (耗时: {elapsed:.2f}s)")
                            return {
                                "case_id": case.case_id,
                                "success": True,
                                "elapsed": elapsed,
                                "result": evaluation_result,
                                "status_code": response.status
                            }
                        except json.JSONDecodeError as e:
                            print(f"⚠️ {case.case_id}: JSON解析失败 - {str(e)}")
                            return {
                                "case_id": case.case_id,
                                "success": False,
                                "elapsed": elapsed,
                                "error": f"JSON解析失败: {str(e)}",
                                "raw_response": ai_response,
                                "status_code": response.status
                            }
                    else:
                        error_text = await response.text()
                        print(f"❌ {case.case_id}: API请求失败 (状态码: {response.status}, 耗时: {elapsed:.2f}s)")
                        return {
                            "case_id": case.case_id,
                            "success": False,
                            "elapsed": elapsed,
                            "error": error_text,
                            "status_code": response.status
                        }
                        
            except Exception as e:
                elapsed = time.time() - start_time
                print(f"❌ {case.case_id}: 异常 - {str(e)} (耗时: {elapsed:.2f}s)")
                return {
                    "case_id": case.case_id,
                    "success": False,
                    "elapsed": elapsed,
                    "error": str(e)
                }

    async def run_concurrent(self, cases: List[TestCase]) -> List[Dict]:
        """
        并发运行所有测试用例
        
        Args:
            cases: 测试用例列表
            
        Returns:
            所有测试结果
        """
        print(f"\n🚀 开始并发测试 (最大并发数: {self.max_concurrent})...\n")
        
        connector = aiohttp.TCPConnector(limit=self.max_concurrent)
        async with aiohttp.ClientSession(connector=connector) as session:
            # 创建所有任务
            tasks = []
            for case in cases:
                task = self.submit_single(session, case)
                tasks.append(task)
            
            # 并发执行所有任务
            self.results = await asyncio.gather(*tasks)
            
            return self.results

    def print_summary(self):
        """打印测试结果汇总"""
        if not self.results:
            print("\n📊 暂无测试结果")
            return
        
        total = len(self.results)
        success = sum(1 for r in self.results if r.get("success"))
        failed = total - success
        
        # 计算统计信息
        times = [r.get("elapsed", 0) for r in self.results if r.get("elapsed")]
        avg_time = sum(times) / len(times) if times else 0
        max_time = max(times) if times else 0
        min_time = min(times) if times else 0
        
        print("\n" + "="*50)
        print("📊 测试结果汇总")
        print("="*50)
        print(f"总测试数: {total}")
        print(f"✅ 成功: {success}")
        print(f"❌ 失败: {failed}")
        print(f"成功率: {(success/total*100):.1f}%" if total > 0 else "N/A")
        print(f"\n⏱️  耗时统计:")
        print(f"  平均: {avg_time:.2f}s")
        print(f"  最大: {max_time:.2f}s")
        print(f"  最小: {min_time:.2f}s")
        
        # 显示失败的用例
        if failed > 0:
            print(f"\n❌ 失败的用例:")
            for r in self.results:
                if not r.get("success"):
                    case_id = r.get("case_id", "未知")
                    error = r.get("error", "未知错误")
                    status = r.get("status_code", "")
                    print(f"  {case_id}: {error} (HTTP {status})")

    def save_results(self, filename: str = None):
        """保存结果到文件"""
        if not filename:
            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            filename = f"ai_test_results_{timestamp}.json"
        
        with open(filename, 'w', encoding='utf-8') as f:
            json.dump({
                "timestamp": datetime.now().isoformat(),
                "total": len(self.results),
                "success_count": sum(1 for r in self.results if r.get("success")),
                "failed_count": sum(1 for r in self.results if not r.get("success")),
                "results": self.results
            }, f, ensure_ascii=False, indent=2)
        
        print(f"\n💾 结果已保存到: {filename}")

async def main():
    """主函数"""
    print("🔧 DeepSeek AI代码评价测试脚本")
    print("="*50)
    # 配置参数
    MAX_CONCURRENT = 25  # 最大并发数

    # 创建测试器
    tester = DirectAITester(
        max_concurrent=MAX_CONCURRENT
    )

    try:
        # 加载所有测试用例
        cases = tester.load_all_cases("test_cases")
        
        if not cases:
            print("❌ 没有找到有效的测试用例")
            return
        
        print(f"\n📋 共找到 {len(cases)} 个测试用例")
        
        # 确认是否继续
        response = input("\n是否开始测试？(y/n): ")
        if response.lower() != 'y':
            print("测试已取消")
            return
        
        # 运行测试（并发执行）
        start_time = time.time()
        await tester.run_concurrent(cases)
        total_time = time.time() - start_time
        
        # 打印汇总
        tester.print_summary()
        print(f"\n⏱️  总耗时: {total_time:.2f}s")
        
        # 保存结果
        tester.save_results()
        
    except KeyboardInterrupt:
        print("\n\n⚠️ 测试被用户中断")
    except Exception as e:
        print(f"\n❌ 测试失败: {str(e)}")
        import traceback
        traceback.print_exc()

if __name__ == "__main__":
    asyncio.run(main())