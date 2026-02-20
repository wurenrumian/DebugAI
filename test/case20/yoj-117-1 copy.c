/*
1. 优秀
2. 优秀
3. 待改进，没有使用记忆化导致超时
4. 优秀
*/
#include<stdio.h>
#define MAX_N 101 // 定义最大层数，避免数组越界
int n;
int tree[MAX_N][MAX_N]; 

// 递归函数：从第i层第j个位置（0起始）爬到顶层（i=0）的最大桃子数
int maxpeach(int i, int j) {
    // 终止条件：到达顶层（第0层），返回当前位置桃子数
    if (i == 0) {
        return tree[i][j];
    }
    int max;
    if (j == 0) {
        // 最左侧位置：只能来自上一层（i-1）的j=0
        max = maxpeach(i-1, j);
    } else if (j == i) {
        // 最右侧位置：只能来自上一层（i-1）的j-1（因为第i层有i+1个元素，j最大为i）
        max = maxpeach(i-1, j-1);
    } else {
        // 中间位置：取上一层两个路径的最大值
        int left = maxpeach(i-1, j-1);
        int right = maxpeach(i-1, j);
        max = (left > right) ? left : right;
    }
    return tree[i][j] + max;
}

int main() {
    scanf("%d", &n);
    // 修正输入逻辑：第i层（0起始）有i+1个桃子，只读取i+1个数据
    for (int i = 0; i < n; i++) {
        for (int j = 0; j <= i; j++) { // 关键修正：j从0到i（共i+1个数据）
            scanf("%d", &tree[i][j]);
        }
    }
    
    int max_result = 0;
    // 最底层是第n-1层，有n个位置（j从0到n-1）
    for (int j = 0; j < n; j++) {
        int current = maxpeach(n-1, j);
        if (current > max_result) {
            max_result = current;
        }
    }
    printf("%d", max_result);
    return 0;
}