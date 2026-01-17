# ai-python/tests/test_debugger.py
import pytest
import asyncio
import sys
sys.path.append('.')  # 将当前目录添加到Python路径

from data import CodeSubmission, TaskType, TestPoint
from debugger import CodeDebugger

@pytest.mark.asyncio
async def test_debug_runtime_error():
    submission = CodeSubmission(
        code="""
#include <stdio.h>
int findMax(int arr[], int n) {
    int result = arr[0];
    for (int i = 1; i <= n; i++) {
        if (arr[i] > result) {
            result = arr[i];
        }
    }
    return result;
}
""",
        problem_description="编写一个程序，找到整数数组中的最大值",
        test_points=[
            TestPoint(input="5\n3 7 2 9 1", status="Runtime Error"),
            TestPoint(input="3\n1 2 3", status="Runtime Error"),
            TestPoint(input="1\n42", status="Accepted")  # n=1时可能不会越界
        ],
        task_type=TaskType.DEBUG
    )
    
    debugger = CodeDebugger()
    result = await debugger.debug(submission)
    
    assert result is not None
    assert len(result.problems) > 0
    assert len(result.suggestions) > 0
    print("C代码调试（运行时错误）测试通过！")

@pytest.mark.asyncio
async def test_debug_time_limit_exceeded():
    submission = CodeSubmission(
        code="""
#include <iostream>
#include <vector>
using namespace std;

int findMax(vector<int>& nums, int index) {
    if (index == 0) return nums[0];
    int maxOfRest = findMax(nums, index - 1);
    return (nums[index] > maxOfRest) ? nums[index] : maxOfRest;
}

int findMax(vector<int>& nums) {
    if (nums.empty()) return -1;
    return findMax(nums, nums.size() - 1);
}
""",
        problem_description="编写一个高效的C++函数，找到整数数组中的最大值",
        test_points=[
            TestPoint(input="1000\n" + "1 " * 1000, status="Time Limit Exceeded"),
            TestPoint(input="10\n1 2 3 4 5 6 7 8 9 10", status="Accepted"),
            TestPoint(input="2000\n" + "5 " * 2000, status="Time Limit Exceeded")
        ],
        task_type=TaskType.DEBUG
    )
    
    debugger = CodeDebugger()
    result = await debugger.debug(submission)
    
    assert result is not None
    assert len(result.problems) > 0
    assert len(result.suggestions) > 0
    # 应提到时间复杂度或递归深度问题
    print("C++代码调试（超时错误）测试通过！")

@pytest.mark.asyncio
async def test_debug_memory_limit_exceeded():
    submission = CodeSubmission(
        code="""
#include <stdio.h>
#include <stdlib.h>

int* findMaxAndCreateArray(int arr[], int n) {
    int* result = (int*)malloc(1000000 * sizeof(int));
    if (result == NULL) return NULL;
    
    int maxVal = arr[0];
    for (int i = 1; i < n; i++) {
        if (arr[i] > maxVal) {
            maxVal = arr[i];
        }
    }
    
    for (int i = 0; i < 1000000; i++) {
        result[i] = maxVal;
    }
    
    return result;
}
""",
        problem_description="编写一个C函数，找到数组中的最大值",
        test_points=[
            TestPoint(input="5\n3 7 2 9 1", status="Memory Limit Exceeded"),
            TestPoint(input="10\n1 2 3 4 5 6 7 8 9 10", status="Memory Limit Exceeded")
        ],
        task_type=TaskType.DEBUG
    )
    
    debugger = CodeDebugger()
    result = await debugger.debug(submission)
    
    assert result is not None
    assert len(result.problems) > 0
    assert len(result.suggestions) > 0
    print("C代码调试（内存超限）测试通过！")

@pytest.mark.asyncio
async def test_debug_compile_error():
    submission = CodeSubmission(
        code="""
#include <iostream>
#include <vector>
using namespace std;

int findMax(vector<int>& nums) {
    if (nums.empty()) return -1
    int maxNum = nums[0]
    for (int i = 1; i < nums.size(); i++) {
        if (nums[i] > maxNum) {
            maxNum = nums[i]
        }
    }
    return maxNum
}
""",
        problem_description="编写一个C++函数，找到整数数组中的最大值",
        test_points=[
            TestPoint(input="5\n3 7 2 9 1", status="Compile Error"),
            TestPoint(input="3\n1 2 3", status="Compile Error")
        ],
        task_type=TaskType.DEBUG
    )
    
    debugger = CodeDebugger()
    result = await debugger.debug(submission)
    
    assert result is not None
    assert len(result.problems) > 0
    assert len(result.suggestions) > 0
    print("C++代码调试（编译错误）测试通过！")

if __name__ == "__main__":
    asyncio.run(test_debug_memory_limit_exceeded())