#include <stdio.h>
#include <string.h>
#include <stdlib.h>

#define MAX_LEN 100

// 大数乘法函数
void multiply_big_numbers(char *a, char *b, char *result) {
    int len1 = strlen(a);
    int len2 = strlen(b);
    int *res = (int *)calloc(len1 + len2, sizeof(int));
    
    // 从低位到高位逐位相乘
    for (int i = len1 - 1; i >= 0; i--) {
        for (int j = len2 - 1; j >= 0; j--) {
            int mul = (a[i] - '0') * (b[j] - '0');
            int sum = res[i + j + 1] + mul;
            
            res[i + j + 1] = sum % 10;
            res[i + j] += sum / 10;
        }
    }
    
    // 将结果转换为字符串
    int idx = 0;
    int start = 0;
    
    // 跳过前导零
    while (start < len1 + len2 && res[start] == 0) {
        start++;
    }
    
    // 如果所有位都是0，则结果为0
    if (start == len1 + len2) {
        result[idx++] = '0';
    } else {
        for (int i = start; i < len1 + len2; i++) {
            result[idx++] = res[i] + '0';
        }
    }
    
    result[idx] = '\0';
    free(res);
}

int main() {
    char input1[MAX_LEN], input2[MAX_LEN], op[2];
    
    // 读取输入为字符串，以支持大数
    if (scanf("%s %s %s", input1, input2, op) != 3) {
        printf("NO\n");
        return 0;
    }
    
    // 验证输入是否为有效数字
    for (int i = 0; input1[i]; i++) {
        if (input1[i] < '0' || input1[i] > '9') {
            printf("NO\n");
            return 0;
        }
    }
    for (int i = 0; input2[i]; i++) {
        if (input2[i] < '0' || input2[i] > '9') {
            printf("NO\n");
            return 0;
        }
    }
    
    // 根据运算符进行计算
    switch (op[0]) {
        case '+': {
            // 对于加法，可以尝试用long long，如果溢出再转大数
            long long num1 = atoll(input1);
            long long num2 = atoll(input2);
            long long result = num1 + num2;
            printf("%lld\n", result);
            break;
        }
        case '-': {
            long long num1 = atoll(input1);
            long long num2 = atoll(input2);
            long long result = num1 - num2;
            printf("%lld\n", result);
            break;
        }
        case '*': {
            // 乘法最容易溢出，直接使用大数乘法
            char result[MAX_LEN * 2];
            multiply_big_numbers(input1, input2, result);
            printf("%s\n", result);
            break;
        }
        case '/': {
            long long num1 = atoll(input1);
            long long num2 = atoll(input2);
            if (num2 == 0) {
                printf("NO\n");
            } else {
                printf("%lld\n", num1 / num2);
            }
            break;
        }
        case '%': {
            long long num1 = atoll(input1);
            long long num2 = atoll(input2);
            if (num2 == 0) {
                printf("NO\n");
            } else {
                printf("%lld\n", num1 % num2);
            }
            break;
        }
        default:
            printf("NO\n");
            break;
    }
    
    return 0;
}