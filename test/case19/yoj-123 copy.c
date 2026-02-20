/*
1. 合格
2. 待改进，对于函数返回的掌握不好，以及同名变量的使用
3. 合格，有冗余计算
4. 优秀
*/
#include <stdio.h>

int step;

int move(char A, char B, char C, int n, int step) {
    if (n == 1) {
        step += 1;
        printf ("[step %d]\t move plate %d#\t from %c to %c\n", step, n, A, C);
    }
    step = move(A, C, B, n - 1, step);
    step += 1;
    printf ("[step %d]\t move plate %d#\t from %c to %c\n", step, n, A, C);
    int total = move(B, A, C, n - 1, step);
}

int main () {
    int n;
    scanf ("%d", &n);
    move('a', 'b', 'c', n, 0);
    printf ("%d\n", step);
    return 0;
}