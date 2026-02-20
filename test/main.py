# main.py
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
    
class ConcurrentTester:
    def __init__(self, base_url: str = "http://localhost:5173", 
                 token: str = None, 
                 max_concurrent: int = 5):
        """
        初始化并发测试器
        
        Args:
            base_url: 基础URL
            token: Bearer认证token
            max_concurrent: 最大并发数
        """
        self.base_url = base_url
        self.api_url = f"{base_url}/api/v1/ai/evaluate"
        self.token = token or self._get_token_from_env()
        self.max_concurrent = max_concurrent
        self.results = []
        
    def _get_token_from_env(self) -> Optional[str]:
        """从环境变量获取token"""
        return os.environ.get("AUTH_TOKEN")
    
    def set_token(self, token: str):
        """设置认证token"""
        self.token = token
        
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
            
            # 读取题目描述（假设是txt文件，可能叫"题目描述.txt"或类似）
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
    
    def load_all_cases(self, test_dir: str = "test") -> List[TestCase]:
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
        # 遍历所有case文件夹（假设命名如case1, case2, ...）
        for item in sorted(test_path.iterdir()):
            if item.is_dir() and item.name.startswith("case"):
                print(f"📂 读取 {item.name}...")
                case = self.read_case_folder(item)
                if case:
                    cases.append(case)
        
        print(f"\n✅ 成功加载 {len(cases)} 个测试用例")
        return cases
    
    async def submit_single(self, session: aiohttp.ClientSession, 
                           case: TestCase, 
                           student_id: str = "2025201717") -> Dict:
        """
        提交单个测试用例
        
        Args:
            session: aiohttp会话
            case: 测试用例
            student_id: 学生ID
            
        Returns:
            响应结果
        """
        start_time = time.time()
        
        # 构建请求数据
        request_data = {
            "student_id": student_id,
            "conversation_id": f"eval_{case.case_id}_{int(time.time())}",
            "code": case.code,
            "problem_description": case.problem_description,
            "test_points": case.test_points,
            "task_type": "evaluate"
        }
        
        headers = {
            "Accept": "application/json, text/plain, */*",
            "Content-Type": "application/json",
            "Authorization": f"Bearer {self.token}",
            "Origin": self.base_url,
            "Referer": f"{self.base_url}/evaluate"
        }
        
        try:
            async with session.post(self.api_url, 
                                  json=request_data, 
                                  headers=headers) as response:
                elapsed = time.time() - start_time
                
                if response.status == 200:
                    result = await response.json()
                    print(f"✅ {case.case_id}: 成功 (耗时: {elapsed:.2f}s)")
                    return {
                        "case_id": case.case_id,
                        "success": True,
                        "elapsed": elapsed,
                        "result": result,
                        "status_code": response.status
                    }
                else:
                    error_text = await response.text()
                    print(f"❌ {case.case_id}: 失败 (状态码: {response.status}, 耗时: {elapsed:.2f}s)")
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
    
    async def run_concurrent(self, cases: List[TestCase], 
                            student_id: str = "2025201717") -> List[Dict]:
        """
        每隔0.5秒发送一个请求，每批5个，之间间隔60秒
        
        Args:
            cases: 测试用例列表
            student_id: 学生ID
            
        Returns:
            所有测试结果
        """
        print(f"\n🚀 开始顺序测试 (每隔1秒发送一个请求)...\n")
        
        connector = aiohttp.TCPConnector(limit=1)  # 限制连接数
        async with aiohttp.ClientSession(connector=connector) as session:
            self.results = []
            
            for i, case in enumerate(cases):
                print(f"\n📤 正在发送第 {i+1}/{len(cases)} 个请求: {case.case_id}")
                
                # 发送单个请求
                result = await self.submit_single(session, case, student_id)
                self.results.append(result)
                
                # 如果不是最后一个请求，等待0.5秒
                if i < len(cases) - 1:
                    if i%5 == 4:  # 每5个请求后等待60秒
                        print(f"⏳ 已发送一批请求，等待60秒...")
                        await asyncio.sleep(60)
                    else:
                        print(f"⏳ 等待0.5秒后发送下一个请求...")
                        await asyncio.sleep(0.5)
            
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
            filename = f"test_results_{timestamp}.json"
        
        with open(filename, 'w', encoding='utf-8') as f:
            json.dump({
                "timestamp": datetime.now().isoformat(),
                "total": len(self.results),
                "results": self.results
            }, f, ensure_ascii=False, indent=2)
        
        print(f"\n💾 结果已保存到: {filename}")

async def main():
    """主函数"""
    print("🔧 AI代码评价并发测试脚本")
    print("="*50)
    
    # 配置参数
    TOKEN = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6MSwic3R1ZGVudF9pZCI6IjIwMjUyMDE3MTciLCJ1c2VyX3R5cGUiOiJhZG1pbiIsImV4cCI6MTc3MTY1MTU4MywiaWF0IjoxNzcxNTY1MTgzfQ._YS8G8chp8FcCQefSisc-P0obxmlvfXALN2U_gdM13w"  # 请替换为你的token
    MAX_CONCURRENT = 5  # 最大并发数
    STUDENT_ID = "2025201717"  # 学生ID
    
    # 创建测试器
    tester = ConcurrentTester(
        base_url="http://localhost:5173",
        token=TOKEN,
        max_concurrent=MAX_CONCURRENT
    )
    
    try:
        # 加载所有测试用例
        cases = tester.load_all_cases("test")
        
        if not cases:
            print("❌ 没有找到有效的测试用例")
            return
        
        print(f"\n📋 共找到 {len(cases)} 个测试用例")
        
        # 确认是否继续
        response = input("\n是否开始测试？(y/n): ")
        if response.lower() != 'y':
            print("测试已取消")
            return
        
        # 运行测试
        start_time = time.time()
        await tester.run_concurrent(cases, student_id=STUDENT_ID)
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

if __name__ == "__main__":
    asyncio.run(main())