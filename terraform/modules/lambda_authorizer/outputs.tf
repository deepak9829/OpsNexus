output "lambda_invoke_arn" {
  value = aws_lambda_function.authorizer.invoke_arn
}

output "lambda_function_name" {
  value = aws_lambda_function.authorizer.function_name
}

output "lambda_role_arn" {
  value = aws_iam_role.lambda.arn
}

output "lambda_arn" {
  value = aws_lambda_function.authorizer.arn
}
