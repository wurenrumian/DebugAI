import pytest
import asyncio
import sys
sys.path.append('.')  # 将当前目录添加到Python路径

from data import CodeSubmission, TaskType, TestPoint
from evaluator import CodeEvaluator

@pytest.mark.asyncio
async def test_evaluate_code():
    submission = CodeSubmission(
        student_id="2025001",
        conversation_id="001",
        code="""
#include <stdio.h>

int max(int a, int b) {
    return (a > b) ? a : b;
}

int find_max(int arr[], int n) {
    if (n <= 0) return -1;
    int result = arr[0];
    for (int i = 1; i < n; i++) {
        if (arr[i] > result) {
            result = arr[i];
        }
    }
    return result;
}

int main() {
    int arr[] = {5, 2, 9, 1, 7};
    int n = sizeof(arr) / sizeof(arr[0]);
    printf("最大值为: %d\\n", find_max(arr, n));
    return 0;
}
""",
        problem_description="编写一个程序，找到整数数组中的最大值",
        test_points=[
            TestPoint(input="5\n5 2 9 1 7", status="Accepted"),
            TestPoint(input="3\n-1 -5 -3", status="Accepted"),
            TestPoint(input="1\n100", status="Accepted")
        ],
        task_type=TaskType.EVALUATE
    )
    
    evaluator = CodeEvaluator()
    result = await evaluator.evaluate(submission)
    
    assert result is not None
    assert result.student_id == "2025001"
    assert result.conversation_id == "001"
    assert 0 <= result.score <= 100
    print(f"C语言代码评价测试通过！得分: {result.score}")

@pytest.mark.asyncio  
async def test_evaluate_cpp_with_edge_cases():
    submission = CodeSubmission(
        student_id="2025001",
        conversation_id="001",
        code="""
#include <iostream>
#include <vector>
#include <climits>
using namespace std;

int findMax(vector<int>& nums) {
    if (nums.empty()) {
        return INT_MIN;
    }
    int maxNum = nums[0];
    for (int num : nums) {
        if (num > maxNum) {
            maxNum = num;
        }
    }
    return maxNum;
}
""",
        problem_description="编写一个C++函数，找到整数数组中的最大值。需要考虑空数组情况，返回INT_MIN",
        test_points=[
            TestPoint(input="5\n3 7 2 9 1", status="Accepted"),
            TestPoint(input="0", status="Accepted"),  # 空数组
            TestPoint(input="3\n-10 -20 -5", status="Accepted"),
            TestPoint(input="1\n0", status="Accepted")
        ],
        task_type=TaskType.EVALUATE
    )
    
    evaluator = CodeEvaluator()
    result = await evaluator.evaluate(submission)
    
    assert result is not None
    assert result.student_id == "2025001"
    assert result.conversation_id == "001"
    assert 0 <= result.score <= 100
    print(f"C++边界情况测试通过！得分: {result.score}")

@pytest.mark.asyncio
async def test_evaluate_cpp_with_time_limit_exceeded():
    submission = CodeSubmission(
        student_id="2025001",
        conversation_id="001",
        code="""
#include <iostream>
#include <vector>
using namespace std;

int findMax(vector<int>& nums) {
    if (nums.empty()) return -1;
    
    for (int i = 0; i < nums.size(); i++) {
        for (int j = 0; j < nums.size() - 1; j++) {
            if (nums[j] > nums[j+1]) {
                swap(nums[j], nums[j+1]);
            }
        }
    }
    return nums[nums.size()-1];
}
""",
        problem_description="编写一个高效的C++函数，找到整数数组中的最大值",
        test_points=[
            TestPoint(input="1000\n" + "1 " * 1000, status="Accepted"),
            TestPoint(input="10000\n" + "1 " * 10000, status="Time Limit Exceeded"),
            TestPoint(input="5000\n" + "2 " * 5000, status="Accepted")
        ],
        task_type=TaskType.EVALUATE
    )
    
    evaluator = CodeEvaluator()
    result = await evaluator.evaluate(submission)
    
    assert result is not None
    assert result.student_id == "2025001"
    assert result.conversation_id == "001"
    assert 0 <= result.score <= 100
    print(f"C++效率测试通过！得分: {result.score}")

if __name__ == "__main__":
    asyncio.run(test_evaluate_code())