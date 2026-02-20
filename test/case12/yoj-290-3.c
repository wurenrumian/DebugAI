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

    // 如果第一个顾客就没有商品，直接NO
    if (commonSize == 0) {
        printf("NO\n");
        return 0;
    }

    // 处理后续顾客
    for (int k = 1; k < n; k++) {
        int mk;
        scanf("%d", &mk);
        int arr[101];
        for (int i = 0; i < mk; i++) {
            scanf("%d", &arr[i]);
        }

        // 如果当前顾客没有商品，交集肯定为空
        if (mk == 0) {
            commonSize = 0;
            break;
        }

        // 用更安全的方式：直接双重循环检查，避免大数组
        int newSize = 0;
        for (int i = 0; i < commonSize; i++) {
            int found = 0;
            for (int j = 0; j < mk; j++) {
                if (common[i] == arr[j]) {
                    found = 1;
                    break;
                }
            }
            if (found) {
                common[newSize] = common[i];
                newSize++;
            }
        }
        commonSize = newSize;
        
        // 如果交集已经为空，提前结束
        if (commonSize == 0) break;
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