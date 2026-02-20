/*
1. 优秀
2. 合格，10000的假设不正确
3. 合格，有更优的算法选择
4. 优秀
*/
#include <stdio.h>

int main() {
    int n;
    scanf("%d", &n);

    int common[101];  // 存放当前交集
    int commonSize;
    
    // 读取第一个顾客
    int m0;
    scanf("%d", &m0);
    for (int i = 0; i < m0; i++) {
        scanf("%d", &common[i]);
    }
    commonSize = m0;

    // 处理后续顾客
    for (int k = 1; k < n; k++) {
        int mk;
        scanf("%d", &mk);
        int arr[101];
        for (int i = 0; i < mk; i++) {
            scanf("%d", &arr[i]);
        }

        // 标记法：用一个标记数组记录该顾客有哪些商品
        int exist[10001] = {0};  // 假设商品编号不超过10000
        for (int i = 0; i < mk; i++) {
            exist[arr[i]] = 1;
        }

        // 检查common中的每个商品是否在该顾客的商品列表中
        int newSize = 0;
        for (int i = 0; i < commonSize; i++) {
            if (exist[common[i]]) {
                common[newSize] = common[i];
                newSize++;
            }
        }
        commonSize = newSize;
    }

    // 输出结果
    if (commonSize == 0) {
        printf("NO\n");
    } else {
        // 排序（冒泡排序）
        for (int i = 0; i < commonSize - 1; i++) {
            for (int j = 0; j < commonSize - 1 - i; j++) {
                if (common[j] > common[j + 1]) {
                    int temp = common[j];
                    common[j] = common[j + 1];
                    common[j + 1] = temp;
                }
            }
        }
        // 输出
        for (int i = 0; i < commonSize; i++) {
            printf("%d", common[i]);
            if (i < commonSize - 1) printf(" ");
        }
        printf("\n");
    }

    return 0;
}