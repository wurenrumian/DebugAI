"""
手动批量评价脚本（调用真实 DeepSeek API，会产生费用；不属于自动化测试，不进入 Docker 镜像）。

用途：修改评价提示词（prompts.py）后，用一批真实学生代码检查评价质量与等级稳定性。
与旧版 test.py 不同，本脚本直接复用服务中的 CodeEvaluator（同一套提示词、定界转义、
输出校验与重试逻辑），不再维护一份会过时的提示词副本。

用法（在 ai-python/ 目录下）：
    export DEEPSEEK_API_KEY=sk-...
    python scripts/manual_eval.py --cases test_cases --repeat 3

用例目录结构（与旧版 test.py 相同）：
    test_cases/
      case01/
        题目.txt        # 题目描述（第一个 *.txt）
        student.c       # 学生代码（第一个文件名不含 copy 的 *.c）
        tests.json      # 可选，测试点列表 [{"input": "...", "status": "Accepted"}, ...]

输出：控制台汇总 + ai_eval_results_<时间戳>.json（含每个用例每次评价的等级与分析）。
--repeat N 会对每个用例评价 N 次并统计各维度等级是否一致，用于观察评分波动。
"""
import argparse
import asyncio
import json
import sys
import time
from collections import Counter
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Optional

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

from data import CodeSubmission, TestPoint  # noqa: E402
from evaluator import CodeEvaluator, DIMENSIONS  # noqa: E402
from llm_client import LLMError, get_llm_client  # noqa: E402


def read_case(case_path: Path) -> Optional[CodeSubmission]:
    case_id = case_path.name
    desc_files = sorted(case_path.glob("*.txt"))
    code_files = sorted(f for f in case_path.glob("*.c") if "copy" not in f.name.lower())
    if not desc_files or not code_files:
        print(f"[跳过] {case_id}: 缺少题目描述(*.txt)或学生代码(*.c)")
        return None
    test_points: List[TestPoint] = []
    json_files = sorted(case_path.glob("*.json"))
    if json_files:
        raw = json.loads(json_files[0].read_text(encoding="utf-8"))
        test_points = [TestPoint(input=str(tp.get("input", "")), status=str(tp.get("status", "")))
                       for tp in raw]
    return CodeSubmission(
        student_id="manual",
        conversation_id=case_id,
        code=code_files[0].read_text(encoding="utf-8"),
        problem_description=desc_files[0].read_text(encoding="utf-8"),
        test_points=test_points,
    )


async def evaluate_once(evaluator: CodeEvaluator, sub: CodeSubmission) -> Dict:
    start = time.time()
    try:
        result = await evaluator.evaluate(sub)
        return {"success": True, "elapsed": time.time() - start, "result": result.model_dump()}
    except LLMError as e:
        return {"success": False, "elapsed": time.time() - start, "error": f"{type(e).__name__}: {e}"}


async def run(cases_dir: Path, repeat: int) -> List[Dict]:
    cases = [c for c in (read_case(p) for p in sorted(cases_dir.iterdir())
                         if p.is_dir() and p.name.startswith("case")) if c]
    print(f"共加载 {len(cases)} 个用例，每个评价 {repeat} 次（并发上限见 LLM_MAX_CONCURRENCY）")
    evaluator = CodeEvaluator(get_llm_client())
    tasks = [evaluate_once(evaluator, sub) for sub in cases for _ in range(repeat)]
    flat = await asyncio.gather(*tasks)
    results = []
    for i, sub in enumerate(cases):
        runs = flat[i * repeat:(i + 1) * repeat]
        grades = {dim: Counter(r["result"][dim]["grade"] for r in runs if r["success"]) for dim in DIMENSIONS}
        results.append({
            "case_id": sub.conversation_id,
            "runs": runs,
            "grade_distribution": {d: dict(c) for d, c in grades.items()},
            "stable": all(len(c) <= 1 for c in grades.values()),
        })
    return results


def print_summary(results: List[Dict]):
    total_runs = sum(len(r["runs"]) for r in results)
    ok_runs = sum(1 for r in results for run in r["runs"] if run["success"])
    times = [run["elapsed"] for r in results for run in r["runs"]]
    print("=" * 50)
    print(f"评价次数: {total_runs}，成功: {ok_runs}，失败: {total_runs - ok_runs}")
    if times:
        print(f"耗时: 平均 {sum(times) / len(times):.2f}s，最大 {max(times):.2f}s")
    unstable = [r for r in results if not r["stable"]]
    print(f"等级不稳定的用例: {len(unstable)}/{len(results)}")
    for r in unstable:
        print(f"  {r['case_id']}: {r['grade_distribution']}")
    for r in results:
        for run in r["runs"]:
            if not run["success"]:
                print(f"  [失败] {r['case_id']}: {run['error']}")


def main():
    parser = argparse.ArgumentParser(description="使用真实模型批量评价学生代码（手动脚本）")
    parser.add_argument("--cases", default="test_cases", help="用例根目录（包含 case* 子目录）")
    parser.add_argument("--repeat", type=int, default=1, help="每个用例评价次数，用于观察等级波动")
    parser.add_argument("--output", default=None, help="结果 JSON 文件路径")
    args = parser.parse_args()

    if not get_llm_client().configured:
        sys.exit("请先设置 DEEPSEEK_API_KEY")
    cases_dir = Path(args.cases)
    if not cases_dir.is_dir():
        sys.exit(f"用例目录不存在: {cases_dir}")

    start = time.time()
    results = asyncio.run(run(cases_dir, max(1, args.repeat)))
    print_summary(results)
    print(f"总耗时: {time.time() - start:.2f}s")

    output = args.output or f"ai_eval_results_{datetime.now():%Y%m%d_%H%M%S}.json"
    Path(output).write_text(json.dumps({"timestamp": datetime.now().isoformat(), "results": results},
                                       ensure_ascii=False, indent=2), encoding="utf-8")
    print(f"结果已保存到: {output}")


if __name__ == "__main__":
    main()
