/*
1. 优秀
2. 合格，溢出了
3. 合格，运算符非法的判断有冗余
4. 优秀
*/
#include <stdio.h>
int main() {
    int a, b;
    char c;
    scanf("%d %d %c", &a, &b, &c);
    if (c != '+' && c != '-' && c != '*' && c != '/' && c != '%') {
        printf("NO\n");
        return 0;
    }
    if ((c == '/' || c == '%') && b == 0) {
        printf("NO\n");
        return 0;
    }
    int result;
    switch (c) {
        case '+':
            result = a + b;
            break;
        case '-':
            result = a - b;
            break;
        case '*':
            result = a * b;
            break;
        case '/':
            result = a / b;
            break;
        case '%':
            result = a % b;
            break;
        default:
            printf("NO\n");
            return 0;
    }
    printf("%d\n", result);
    return 0;
}