/*
全部优秀
*/
#include<stdio.h>
#define MAX_N 101

int main() {
    int n;
    int tree[MAX_N][MAX_N];
    int dp[MAX_N][MAX_N]; // dp[i][j]表示从底层到(i,j)位置的最大桃子数
    
    scanf("%d", &n);
    
    // 读取数据
    for (int i = 0; i < n; i++) {
        for (int j = 0; j <= i; j++) {
            scanf("%d", &tree[i][j]);
        }
    }
    
    // 初始化底层
    for (int j = 0; j < n; j++) {
        dp[n-1][j] = tree[n-1][j];
    }
    
    // 自底向上计算
    for (int i = n-2; i >= 0; i--) {
        for (int j = 0; j <= i; j++) {
            // 每个位置可以从下一层的j或j+1位置上来
            int left = dp[i+1][j];
            int right = dp[i+1][j+1];
            dp[i][j] = tree[i][j] + ((left > right) ? left : right);
        }
    }
    
    // 顶层就是结果
    printf("%d", dp[0][0]);
    return 0;
}